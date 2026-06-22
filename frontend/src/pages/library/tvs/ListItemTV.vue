<template>
  <q-card flat class="tv-card">
    <div class="tv-card__cover">
      <div v-if="!posterInfo?.url" class="tv-card__cover-placeholder"></div>
      <q-img v-else :src="getUrl(posterInfo.url)" class="tv-card__image" no-spinner style="height: 240px" fit="cover" />
    </div>

    <div class="tv-card__body">
      <div class="tv-card__title text-ellipsis-line-2" :title="data.name">{{ data.name }}</div>

      <div class="tv-card__footer">
        <dialog-t-v-detail :data="detailInfo">
          <q-btn
            v-if="hasSubtitleVideoCount > 0"
            color="primary"
            flat
            dense
            icon="closed_caption"
            :label="`${hasSubtitleVideoCount}/${detailInfo.one_video_info.length}`"
            title="已有字幕"
          />
          <q-btn v-else color="grey-5" round flat dense icon="closed_caption" title="没有字幕" />
        </dialog-t-v-detail>

        <div class="tv-card__status" :class="{ 'has-subtitle': hasSubtitleVideoCount > 0 }">
          <span>{{ coverageText }}</span>
        </div>
      </div>
    </div>
  </q-card>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import DialogTVDetail from 'pages/library/tvs/DialogTVDetail';
import LibraryApi from 'src/api/LibraryApi';
import { getUrl, subtitleUploadList } from 'pages/library/use-library';

const props = defineProps({
  data: Object,
});

const posterInfo = ref(null);
const detailInfo = ref(null);

const getPosterInfo = async () => {
  const [res] = await LibraryApi.getTvPoster({
    name: props.data.name,
    main_root_dir_f_path: props.data.main_root_dir_f_path,
    root_dir_path: props.data.root_dir_path,
  });
  posterInfo.value = res;
};

const getDetailInfo = async () => {
  const [res] = await LibraryApi.getTvDetail({
    name: props.data.name,
    main_root_dir_f_path: props.data.main_root_dir_f_path,
    root_dir_path: props.data.root_dir_path,
  });
  detailInfo.value = res;
};

const hasSubtitleVideoCount = computed(
  () => detailInfo.value?.one_video_info.filter((e) => e.sub_f_path_list.length > 0).length
);

const coverageText = computed(() => {
  const total = detailInfo.value?.one_video_info?.length ?? 0;
  const covered = hasSubtitleVideoCount.value ?? 0;

  if (covered === 0 || total === 0) {
    return '待补字幕';
  }
  if (covered >= total) {
    return '已全部覆盖';
  }
  return '已覆盖部分剧集';
});

watch(subtitleUploadList, (val, oldValue) => {
  if (
    detailInfo.value?.one_video_info.some((e) => oldValue.map((f) => f.video_f_path).includes(e.video_f_path)) &&
    !detailInfo.value?.one_video_info.some((e) => val.map((f) => f.video_f_path).includes(e.video_f_path))
  ) {
    getDetailInfo();
  }
});

onMounted(() => {
  getPosterInfo();
  getDetailInfo();
});
</script>

<style lang="scss" scoped>
.tv-card {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  border-radius: 24px;
  border: 1px solid rgba(15, 23, 42, 0.06);
  background: rgba(255, 255, 255, 0.94);
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.tv-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 24px 50px rgba(15, 23, 42, 0.12);
}

.tv-card__cover {
  padding: 8px;
}

.tv-card__cover-placeholder,
.tv-card__image {
  width: 100%;
  height: 240px;
  border-radius: 18px;
  background: linear-gradient(180deg, #f0f4fa 0%, #e5edf8 100%);
}

.tv-card__body {
  display: grid;
  gap: 10px;
  flex: 1 1 auto;
  padding: 0 10px 12px;
}

.tv-card__title {
  min-height: 42px;
  font-size: 15px;
  font-weight: 700;
  line-height: 1.4;
  color: #142033;
}

.tv-card__footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.tv-card__status {
  display: inline-flex;
  align-items: center;
  min-height: 30px;
  padding: 0 10px;
  border-radius: 999px;
  background: #f4f7fb;
  color: #64748b;
  font-size: 12px;
  font-weight: 600;
}

.tv-card__status.has-subtitle {
  background: rgba(43, 182, 115, 0.12);
  color: #1f8b59;
}

.text-ellipsis-line-2 {
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

@media (max-width: 599px) {
  .tv-card__cover-placeholder,
  .tv-card__image {
    height: 220px;
  }

  .tv-card__body {
    gap: 10px;
    padding: 0 10px 12px;
  }

  .tv-card__title {
    min-height: 40px;
    font-size: 14px;
  }
}
</style>
