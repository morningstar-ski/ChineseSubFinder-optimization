import { reactive, ref, watch } from 'vue';
import SettingApi from 'src/api/SettingApi';
import { SystemMessage } from 'src/utils/message';
import { deepCopy } from 'src/utils/common';
import { useAppStatusLoading } from 'src/composables/use-app-status-loading';
import { isRunningInDocker } from 'src/store/systemState';
import { Dialog } from 'quasar';
import { settingsState } from 'src/store/settingsState';

const { startLoading } = useAppStatusLoading();

export const formModel = reactive({});

// 更新emby目录映射
const updateFolderMap = () => {
  const getFolderMap = (folders, maps) =>
    folders.reduce((r, a) => {
      if (Object.keys(maps).includes(a)) {
        r[a] = maps[a];
      } else {
        r[a] = '';
      }
      return r;
    }, {});

  const commonSettings = settingsState.settings?.common_settings;
  const embySettings = settingsState.settings.emby_settings;

  embySettings.movie_paths_mapping = getFolderMap(commonSettings.movie_paths, embySettings.movie_paths_mapping || {});
  embySettings.series_paths_mapping = getFolderMap(
    commonSettings.series_paths,
    embySettings.series_paths_mapping || {}
  );
};

watch(
  () => settingsState.settings,
  () => {
    updateFolderMap();
    Object.assign(formModel, deepCopy(settingsState.settings));
  }
);

export const resetForm = () => {
  Object.assign(formModel, deepCopy(settingsState.settings));
};

const getSettings = async () => {
  const [res, err] = await SettingApi.get();
  if (err !== null) {
    SystemMessage.error(err.message);
    return;
  }
  settingsState.settings = res;
};

export const useSettings = () => {
  getSettings();
};

export const submitting = ref(false);

const validateLLMSubtitleFallback = () => {
  const llm = formModel.experimental_function?.llm_subtitle_fallback;
  if (!llm?.enable) {
    return null;
  }

  const requiredFields = [
    ['服务提供方', llm.provider],
    ['接口地址', llm.base_url],
    ['API key', llm.api_key],
    ['模型', llm.model],
    ['源语言', llm.source_language],
    ['目标语言', llm.target_language],
    ['翻译风格', llm.translate_style],
  ];

  const missingField = requiredFields.find(([, value]) => !String(value ?? '').trim());
  if (missingField) {
    return `${missingField[0]}不能为空`;
  }

  return null;
};

export const submitAll = async () => {
  if (isRunningInDocker.value) {
    const isMoviePathStarsWithMedia = formModel.common_settings.movie_paths.every((path) => path.startsWith('/media'));
    const isSeriesPathStarsWithMedia = formModel.common_settings.series_paths.every((path) =>
      path.startsWith('/media')
    );
    if (!isMoviePathStarsWithMedia || !isSeriesPathStarsWithMedia) {
      Dialog.create({
        title: '请修改相关配置后继续',
        html: true,
        message:
          '软件运行在Docker中，请将基础设置中的电影和电视剧目录修改为 <b>/media</b> 下的目录，否则可能会因为权限问题导致无法正确的加载媒体库',
        persistent: true,
        ok: '确定',
      });
      return;
    }
  }
  const llmValidationError = validateLLMSubtitleFallback();
  if (llmValidationError) {
    SystemMessage.error(llmValidationError);
    return;
  }
  submitting.value = true;
  if (formModel.experimental_function?.local_chrome_settings) {
    formModel.experimental_function.local_chrome_settings.configured = true;
  }
  const [, err] = await SettingApi.update(formModel);
  submitting.value = false;
  if (err !== null) {
    SystemMessage.error(err.message);
    return;
  }
  settingsState.settings = { ...settingsState.settings, ...deepCopy(formModel) };
  SystemMessage.success('保存成功');
  startLoading();
};

/**
 * 获取导出的settings
 * @param includeSensitive
 * @returns {any}
 */
export const getExportSettings = (includeSensitive = false) => {
  const data = deepCopy(settingsState.settings);
  if (!includeSensitive) {
    delete data.user_info;
    delete data.advanced_settings.proxy_settings;
    delete data.common_settings.threads;
    delete data.emby_settings.api_key;
    delete data.emby_settings.address_url;
    delete data.experimental_function.api_key_settings;
    delete data.experimental_function.llm_subtitle_fallback.api_key;
  }
  return data;
};
