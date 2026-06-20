<template>
  <div>
    <q-list class="settings-panel-list" dense>
      <q-item>
        <q-item-section>
          <q-item-label>自动转换字幕文件编码</q-item-label>
          <q-item-label caption> 自动转换到目标编码。如果不是特殊情况，不建议开启，仅对新下载字幕生效。 </q-item-label>
          <q-item v-if="form.auto_change_sub_encode.enable">
            <q-item-section avatar top>
              <q-radio
                v-for="(v, k) in DESC_ENCODE_TYPE_NAME_MAP"
                :key="k"
                :label="v"
                v-model="form.auto_change_sub_encode.des_encode_type"
                :val="~~k"
              />
            </q-item-section>
          </q-item>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.auto_change_sub_encode.enable" />
        </q-item-section>
      </q-item>

      <q-separator spaced inset />

      <q-item tag="label" :disable="!isChsChtChangerEnable" v-ripple>
        <q-item-section>
          <q-item-label>简繁字幕互转</q-item-label>
          <q-item-label caption> 需要先开启自动转换字幕文件编码，并设置为 UTF-8，否则无法启用和生效。 </q-item-label>
          <q-item v-if="form.chs_cht_changer.enable">
            <q-item-section avatar top>
              <q-radio
                :disable="!isChsChtChangerEnable"
                v-for="(v, k) in AUTO_CONVERT_LANG_NAME_MAP"
                :key="k"
                :label="v"
                v-model="form.chs_cht_changer.des_chinese_language_type"
                :val="~~k"
              />
            </q-item-section>
          </q-item>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle :disable="!isChsChtChangerEnable" v-model="form.chs_cht_changer.enable" />
        </q-item-section>
      </q-item>

      <q-separator spaced inset />

      <q-item>
        <q-item-section>
          <q-item-label>远程 Chrome</q-item-label>
          <q-item-label caption>
            只在你明确需要把浏览器抓取放到另一台机器执行时才启用。 默认字幕链路不依赖这组配置。
          </q-item-label>
        </q-item-section>
        <q-item-section avatar>
          <q-toggle v-model="form.remote_chrome_settings.enable" />
        </q-item-section>
      </q-item>

      <template v-if="form.remote_chrome_settings.enable">
        <q-item>
          <q-item-section>
            <q-item-label>远程 Chrome DevTools 地址</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input
              v-model="form.remote_chrome_settings.remote_docker_url"
              placeholder="ws://192.168.xx.xx:9222"
              standout
              dense
              :rules="[(val) => (form.remote_chrome_settings.enable && !!val) || '不能为空']"
            />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>远程用户数据目录</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input
              v-model="form.remote_chrome_settings.remote_user_data_dir"
              placeholder="/mnt/share/tmp"
              standout
              dense
            />
          </q-item-section>
        </q-item>
      </template>

      <q-separator spaced inset />

      <q-item>
        <q-item-section>
          <q-item-label>{{ localChromeLabel }}</q-item-label>
          <q-item-label caption>
            {{ localChromeCaption }}
          </q-item-label>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.local_chrome_settings.enabled" />
        </q-item-section>
      </q-item>

      <template v-if="form.local_chrome_settings.enabled">
        <q-item>
          <q-item-section>
            <q-item-label caption>
              {{ localChromeEnabledNote }}
            </q-item-label>
          </q-item-section>
        </q-item>
      </template>

      <q-separator spaced inset />

      <q-item>
        <q-item-section>
          <q-item-label>LLM 英文转中文字幕回退链</q-item-label>
          <q-item-label caption>
            独立控制英文字幕翻译回退。只会在原生中文字幕候选失败后触发， 且仅对单字幕保存模式生效。
          </q-item-label>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.llm_subtitle_fallback.enable" />
        </q-item-section>
      </q-item>

      <template v-if="form.llm_subtitle_fallback.enable">
        <q-item v-if="isRunningInDocker">
          <q-item-section>
            <q-item-label caption>
              Docker 一键部署会自动使用镜像内置的 Python、ddddocr 和 bundled Subflow， 不需要再手动配置运行时路径。
            </q-item-label>
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>服务提供方</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input v-model="form.llm_subtitle_fallback.provider" standout dense />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>接口地址</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input
              v-model="form.llm_subtitle_fallback.base_url"
              placeholder="兼容 OpenAI 的接口地址，可留空走 Gemini 原生"
              standout
              dense
            />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>API key</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input
              v-model="form.llm_subtitle_fallback.api_key"
              type="password"
              placeholder="留空则回退到 subflow 本地配置或环境变量"
              standout
              dense
            />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>模型</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input v-model="form.llm_subtitle_fallback.model" standout dense />
          </q-item-section>
        </q-item>

        <template v-if="showLLMRuntimeInputs">
          <q-item>
            <q-item-section>
              <q-item-label>Python 可执行文件</q-item-label>
            </q-item-section>
            <q-item-section avatar>
              <q-input
                v-model="form.llm_subtitle_fallback.python_executable"
                placeholder="留空则自动探测运行时 Python"
                standout
                dense
              />
            </q-item-section>
          </q-item>

          <q-item>
            <q-item-section>
              <q-item-label>Subflow 根目录</q-item-label>
            </q-item-section>
            <q-item-section avatar>
              <q-input
                v-model="form.llm_subtitle_fallback.subflow_root_dir"
                placeholder="留空则自动使用 bundled 或默认路径"
                standout
                dense
              />
            </q-item-section>
          </q-item>

          <q-item>
            <q-item-section>
              <q-item-label>日志目录</q-item-label>
            </q-item-section>
            <q-item-section avatar>
              <q-input
                v-model="form.llm_subtitle_fallback.log_dir"
                placeholder="留空则自动使用默认日志目录"
                standout
                dense
              />
            </q-item-section>
          </q-item>
        </template>

        <q-item>
          <q-item-section>
            <q-item-label>源语言</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input v-model="form.llm_subtitle_fallback.source_language" standout dense />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>目标语言</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input v-model="form.llm_subtitle_fallback.target_language" standout dense />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>翻译风格</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input v-model="form.llm_subtitle_fallback.translate_style" placeholder="可留空" standout dense />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>仅在无中文候选时触发</q-item-label>
            <q-item-label caption> 默认就是这个策略，这里保留显式开关。 </q-item-label>
          </q-item-section>
          <q-item-section avatar top>
            <q-toggle v-model="form.llm_subtitle_fallback.only_when_no_chinese_candidate" />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>保留英文源字幕副本</q-item-label>
          </q-item-section>
          <q-item-section avatar top>
            <q-toggle v-model="form.llm_subtitle_fallback.keep_english_source_copy" />
          </q-item-section>
        </q-item>
      </template>

      <q-separator spaced inset />

      <q-item>
        <q-item-section>
          <q-item-label>API key</q-item-label>
          <q-item-label caption>
            本程序提供了一组面向开发者的接口，通过 API key 鉴权。
            <br />
            接口说明见
            <a :href="apiKeyDocUrl" class="text-primary" target="_blank"> 开发文档 </a>
            。
          </q-item-label>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.api_key_settings.enabled" />
        </q-item-section>
      </q-item>

      <template v-if="form.api_key_settings.enabled">
        <q-item>
          <q-btn label="重新生成密钥" color="primary" size="sm" @click="generateApiKey" />
        </q-item>
        <q-item class="q-mt-sm">
          <q-item-section>
            <q-input
              v-model="form.api_key_settings.key"
              standout
              dense
              :rules="[(val) => !!val || '不能为空']"
              readonly
            >
              <template #append>
                <copy-to-clipboard-btn v-if="form.api_key_settings.key" :text="form.api_key_settings.key" size="sm" />
              </template>
            </q-input>
          </q-item-section>
        </q-item>
      </template>
    </q-list>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import { toRefs } from '@vueuse/core';
import CopyToClipboardBtn from 'components/CopyToClipboardBtn';
import { formModel } from 'pages/settings/use-settings';
import {
  AUTO_CONVERT_LANG_NAME_MAP,
  DESC_ENCODE_TYPE_NAME_MAP,
  DESC_ENCODE_TYPE_UTF8,
} from 'src/constants/SettingConstants';
import { isRunningInDocker } from 'src/store/systemState';

const { experimental_function: form } = toRefs(formModel);

const isChsChtChangerEnable = computed(
  () =>
    formModel.experimental_function.auto_change_sub_encode?.enable &&
    formModel.experimental_function.auto_change_sub_encode?.des_encode_type === DESC_ENCODE_TYPE_UTF8
);

const showLLMRuntimeInputs = computed(() => !isRunningInDocker.value);
const localChromeLabel = computed(() => (isRunningInDocker.value ? '内置 Chrome' : '本地 Chrome'));
const localChromeCaption = computed(() =>
  isRunningInDocker.value
    ? 'Docker 一键部署默认直接启动镜像内置 Chrome。'
    : '开启后使用本机可用的 Chrome 处理依赖浏览器的流程。'
);
const localChromeEnabledNote = computed(() =>
  isRunningInDocker.value ? '当前环境不需要额外安装浏览器，也不需要填写路径。' : '当前环境会直接使用本机可用的 Chrome。'
);

const apiKeyDocUrl = [
  'https://github.com/ChineseSubFinder/ChineseSubFinder/blob/docs/DesignFile/',
  'ApiKey%E8%AE%BE%E8%AE%A1/ApiKey%E8%AE%BE%E8%AE%A1.md',
].join('');

const generateUuid = () =>
  'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    // eslint-disable-next-line no-bitwise
    const r = (Math.random() * 16) | 0;
    // eslint-disable-next-line no-bitwise
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });

const generateApiKey = () => {
  const uuid = generateUuid();
  formModel.experimental_function.api_key_settings.key = uuid;
};
</script>
