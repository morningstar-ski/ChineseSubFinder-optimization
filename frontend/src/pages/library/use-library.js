import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import LibraryApi from 'src/api/LibraryApi';
import { SystemMessage } from 'src/utils/message';
import { until } from '@vueuse/core';
import config from 'src/config';
import { LocalStorage } from 'quasar';
import { useSettings } from 'pages/settings/use-settings';

export const getUrl = (basePath) => config.BACKEND_URL + basePath.split(/\/|\\/).join('/');

export const coverRule = ref(LocalStorage.getItem('coverRule') ?? 'poster.jpg');

export const originMovies = ref([]);
export const originTvs = ref([]);

const movies = computed(() =>
  originMovies.value.map((movie) => ({
    ...movie,
  }))
);

const tvs = computed(() =>
  originTvs.value.map((tv) => ({
    ...tv,
  }))
);

export const libraryRefreshStatus = ref(null);
export const subtitleUploadList = ref([]);
export const subtitleJobResults = ref({});

export const refreshCacheLoading = computed(() => libraryRefreshStatus.value === 'running');

let getRefreshStatusTimer = null;

const sleep = (ms) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

export const getTimelineFixJobKey = ({ videoPath, subPath }) => `fix_timeline_only|${videoPath}|${subPath}`;

export const getLibraryRefreshStatus = async () => {
  const [res] = await LibraryApi.getRefreshStatus();
  libraryRefreshStatus.value = res.status;
};

export const getLibraryList = async () => {
  const [res, err] = await LibraryApi.getList();
  if (err !== null) {
    SystemMessage.error(err.message);
  } else {
    originMovies.value = res.movie_infos_v2;
    originTvs.value = res.season_infos_v2;
  }
};

export const checkLibraryRefreshStatus = async () => {
  libraryRefreshStatus.value = null;
  await getLibraryRefreshStatus();
  getRefreshStatusTimer = setInterval(() => {
    getLibraryRefreshStatus();
  }, 1000);
  await until(libraryRefreshStatus).toBe('stopped');
  clearInterval(getRefreshStatusTimer);
  getRefreshStatusTimer = null;
  await getLibraryList();
};

export const refreshLibrary = async () => {
  const [, err] = await LibraryApi.refreshLibrary();
  if (err !== null) {
    SystemMessage.error(err.message);
  } else {
    await checkLibraryRefreshStatus();
    SystemMessage.success('更新媒体库缓存成功');
  }
};

export const getSubtitleUploadList = async () => {
  const [res] = await LibraryApi.getSubTitleQueueList();
  subtitleUploadList.value = res.jobs;
};

export const getManualSubtitleJobResult = async ({ videoPath, subPath, mode = '' }) => {
  const [res, err] = await LibraryApi.getManualSubtitleResult({
    video_f_path: videoPath,
    sub_f_path: subPath,
    mode,
  });
  if (err !== null) {
    return [null, err];
  }
  return [res?.message ?? '', null];
};

export const getTimelineFixJobStatus = ({ videoPath, subPath }) => {
  const isQueued = subtitleUploadList.value.some(
    (item) => item.video_f_path === videoPath && item.sub_f_path === subPath && item.mode === 'fix_timeline_only'
  );
  if (isQueued) {
    return 'pending';
  }
  return subtitleJobResults.value[getTimelineFixJobKey({ videoPath, subPath })] ?? '';
};

const waitForTimelineFixResult = async ({ videoPath, subPath, deadline }) => {
  if (Date.now() >= deadline) {
    return { status: 'timeout', message: '' };
  }

  await sleep(1500);
  await getSubtitleUploadList();

  if (getTimelineFixJobStatus({ videoPath, subPath }) === 'pending') {
    return waitForTimelineFixResult({ videoPath, subPath, deadline });
  }

  const [result, resultErr] = await getManualSubtitleJobResult({
    videoPath,
    subPath,
    mode: 'fix_timeline_only',
  });
  if (resultErr !== null) {
    return { status: 'failed', message: resultErr.message };
  }
  if (result === 'ok') {
    return { status: 'done', message: '' };
  }
  if (result) {
    return { status: 'failed', message: result };
  }

  return waitForTimelineFixResult({ videoPath, subPath, deadline });
};

export const useLibrary = () => {
  useSettings();

  const getSubtitleUploadListTimer = setInterval(() => {
    getSubtitleUploadList();
  }, 5000);

  onMounted(() => {
    getLibraryList();
    getLibraryRefreshStatus();
    getSubtitleUploadList();
    checkLibraryRefreshStatus();
  });

  onBeforeUnmount(() => {
    clearInterval(getRefreshStatusTimer);
    clearInterval(getSubtitleUploadListTimer);
  });

  return {
    movies,
    tvs,
    refreshLibrary,
    refreshCacheLoading,
  };
};

export const doFixSubtitleTimeline = async ({ videoPath, subPath }) => {
  const [, err] = await LibraryApi.fixSubtitleTimeline({
    video_f_path: videoPath,
    sub_f_path: subPath,
  });
  if (err !== null) {
    SystemMessage.error(err.message);
    return;
  }

  const jobKey = getTimelineFixJobKey({ videoPath, subPath });
  subtitleJobResults.value = {
    ...subtitleJobResults.value,
    [jobKey]: 'pending',
  };

  SystemMessage.info('已加入时间轴校准队列，正在等待处理结果。', {
    timeout: 3000,
  });
  await getSubtitleUploadList();

  const { status, message } = await waitForTimelineFixResult({
    videoPath,
    subPath,
    deadline: Date.now() + 120000,
  });

  subtitleJobResults.value = {
    ...subtitleJobResults.value,
    [jobKey]: status,
  };

  if (status === 'done') {
    SystemMessage.success('时间轴校准完成。', {
      timeout: 2500,
    });
    await getLibraryList();
    return;
  }

  if (status === 'failed') {
    SystemMessage.error(`时间轴校准失败：${message}`);
    return;
  }

  SystemMessage.warn('时间轴校准结果等待超时，请稍后刷新字幕列表确认。', {
    timeout: 3000,
  });
};

export const checkIsVideoLocked = async (videoInfo) => {
  const [res] = await LibraryApi.getSkipInfo({
    video_skip_infos: [videoInfo],
  });
  return !!res.is_skips?.[0];
};
