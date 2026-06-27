<template>
  <q-card flat class="movie-card">
    <div class="movie-card__cover">
      <div v-if="!posterInfo?.url" :style="{ height: coverHeight }" class="movie-card__cover-placeholder"></div>
      <q-img
        v-else
        :src="getUrl(posterInfo.url)"
        class="movie-card__image"
        no-spinner
        :style="{ height: coverHeight }"
        fit="cover"
      />

      <div class="movie-card__tools">
        <q-btn
          v-if="hasSubtitle"
          size="sm"
          color="white"
          text-color="dark"
          round
          unelevated
          dense
          icon="closed_caption"
          @click.stop
          title="已有字幕"
        >
          <q-popup-proxy>
            <q-list dense class="movie-card__subtitle-list">
              <q-item v-for="(item, index) in detialInfo.sub_url_list" :key="item">
                <q-item-section side>{{ index + 1 }}.</q-item-section>

                <q-item-section class="overflow-hidden ellipsis" :title="item.split(/\/|\\/).pop()">
                  <a class="text-primary" :href="getUrl(item)" target="_blank">{{ item.split(/\/|\\/).pop() }}</a>
                </q-item-section>
              </q-item>
            </q-list>
          </q-popup-proxy>
        </q-btn>

        <q-btn
          v-else
          color="grey-4"
          size="sm"
          round
          unelevated
          dense
          icon="closed_caption"
          @click.stop
          title="没有字幕"
        />
      </div>
    </div>

    <div class="movie-card__body">
      <div class="movie-card__title text-ellipsis-line-2" :title="data.name">{{ data.name }}</div>

      <div class="movie-card__meta">
        <div class="movie-card__status" :class="{ 'has-subtitle': hasSubtitle }">
          <q-icon :name="hasSubtitle ? 'subtitles' : 'subtitles_off'" size="16px" />
          <span>{{ hasSubtitle ? '已匹配字幕' : '待补字幕' }}</span>
        </div>

        <div v-if="hasSubtitle" class="movie-card__meta-actions">
          <btn-dialog-preview-video size="sm" :subtitle-url-list="detialInfo?.sub_url_list" :path="data.video_f_path" />

          <q-btn color="primary" icon="av_timer" flat dense round title="校时间轴" @click.stop>
            <q-popup-proxy anchor="top right">
              <q-list dense class="movie-card__subtitle-list">
                <q-item
                  v-for="(item, index) in detialInfo.sub_url_list"
                  :key="`${item}-fix`"
                  clickable
                  v-ripple
                  v-close-popup
                  @click="
                    doFixSubtitleTimeline({
                      videoPath: data.video_f_path,
                      subPath: detialInfo.sub_f_path_list[index],
                    })
                  "
                >
                  <q-item-section side>{{ index + 1 }}.</q-item-section>
                  <q-item-section class="overflow-hidden ellipsis" :title="item.split(/\/|\\/).pop()">
                    {{ item.split(/\/|\\/).pop() }}
                  </q-item-section>
                  <q-item-section side>
                    <q-icon name="av_timer" color="primary" size="18px" />
                  </q-item-section>
                </q-item>
              </q-list>
            </q-popup-proxy>
          </q-btn>
        </div>
      </div>

      <div class="movie-card__actions">
        <btn-dialog-search-subtitle :path="props.data.video_f_path" is-movie />
        <btn-upload-subtitle :path="data.video_f_path" dense size="sm" />

        <q-btn
          class="movie-card__action"
          color="primary"
          round
          flat
          dense
          icon="download_for_offline"
          title="加入下载队列"
          @click="downloadSubtitle"
          size="sm"
        />

        <btn-ignore-video :path="props.data.video_f_path" :video-type="VIDEO_TYPE_MOVIE" size="sm" />
      </div>
    </div>
  </q-card>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import LibraryApi from 'src/api/LibraryApi';
import { SystemMessage } from 'src/utils/message';
import { VIDEO_TYPE_MOVIE } from 'src/constants/SettingConstants';
import { useQuasar } from 'quasar';
import { doFixSubtitleTimeline, getUrl, subtitleUploadList } from 'pages/library/use-library';
import BtnIgnoreVideo from 'pages/library/BtnIgnoreVideo';
import BtnUploadSubtitle from 'pages/library/BtnUploadSubtitle';
import BtnDialogPreviewVideo from 'pages/library/BtnDialogPreviewVideo';
import BtnDialogSearchSubtitle from 'pages/library/BtnDialogSearchSubtitle';

const props = defineProps({
  data: Object,
  coverHeight: {
    type: String,
    default: '200px',
  },
});

const $q = useQuasar();

const posterInfo = ref(null);
const detialInfo = ref(null);

const getPosterInfo = async () => {
  const [res] = await LibraryApi.getMoviePoster({
    name: props.data.name,
    main_root_dir_f_path: props.data.main_root_dir_f_path,
    video_f_path: props.data.video_f_path,
  });
  posterInfo.value = res;
};

const getDetailInfo = async () => {
  const [res] = await LibraryApi.getMovieDetail({
    name: props.data.name,
    main_root_dir_f_path: props.data.main_root_dir_f_path,
    video_f_path: props.data.video_f_path,
  });
  detialInfo.value = res;
};

const hasSubtitle = computed(() => (detialInfo.value?.sub_url_list?.length ?? 0) > 0);

const downloadSubtitle = async () => {
  $q.dialog({
    title: '加入下载队列',
    message: '选择下载任务类型',
    options: {
      model: 3,
      type: 'radio',
      items: [
        { label: '插队任务', value: 3 },
        { label: '一次性任务（成功后忽略）', value: 0 },
      ],
    },
    cancel: true,
    persistent: true,
  }).onOk(async (val) => {
    const [, err] = await LibraryApi.downloadSubtitle({
      video_type: VIDEO_TYPE_MOVIE,
      physical_video_file_full_path: props.data.video_f_path,
      task_priority_level: val,
      media_server_inside_video_id: props.data.media_server_inside_video_id,
    });
    if (err !== null) {
      SystemMessage.error(err.message);
    } else {
      SystemMessage.success('已加入下载队列');
    }
  });
};

watch(subtitleUploadList, (val, oldVal) => {
  const prev = oldVal ?? [];
  if (
    (val.find((e) => e.video_f_path === props.data.video_f_path) &&
      !prev.find((e) => e.video_f_path === props.data.video_f_path)) ||
    (!val.find((e) => e.video_f_path === props.data.video_f_path) &&
      prev.find((e) => e.video_f_path === props.data.video_f_path))
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
.movie-card {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  border-radius: 24px;
  border: 1px solid rgba(15, 23, 42, 0.06);
  background: rgba(255, 255, 255, 0.94);
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.movie-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 24px 50px rgba(15, 23, 42, 0.12);
}

.movie-card__cover {
  position: relative;
  padding: 8px;
}

.movie-card__cover-placeholder,
.movie-card__image {
  width: 100%;
  border-radius: 18px;
  background: linear-gradient(180deg, #f0f4fa 0%, #e5edf8 100%);
}

.movie-card__tools {
  position: absolute;
  top: 18px;
  right: 18px;
}

.movie-card__body {
  display: grid;
  gap: 10px;
  flex: 1 1 auto;
  padding: 0 10px 12px;
}

.movie-card__title {
  min-height: 42px;
  font-size: 15px;
  font-weight: 700;
  line-height: 1.4;
  color: #142033;
}

.movie-card__meta,
.movie-card__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.movie-card__meta {
  flex-wrap: wrap;
  justify-content: space-between;
}

.movie-card__meta-actions {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.movie-card__status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 30px;
  padding: 0 10px;
  border-radius: 999px;
  background: #f4f7fb;
  color: #64748b;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
  flex: 0 0 auto;
}

.movie-card__status.has-subtitle {
  background: rgba(43, 182, 115, 0.12);
  color: #1f8b59;
}

.movie-card__actions {
  flex-wrap: wrap;
  margin-top: auto;
}

.movie-card__subtitle-list {
  min-width: 300px;
}

.text-ellipsis-line-2 {
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

@media (max-width: 599px) {
  .movie-card__body {
    gap: 10px;
    padding: 0 10px 12px;
  }

  .movie-card__title {
    min-height: 40px;
    font-size: 14px;
  }
}
</style>
