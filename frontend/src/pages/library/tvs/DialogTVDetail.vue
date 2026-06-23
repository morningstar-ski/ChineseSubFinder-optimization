<template>
  <span @click.stop="visible = true">
    <slot></slot>
  </span>

  <q-dialog v-model="visible">
    <q-card style="width: 900px; max-width: 900px">
      <q-card-section>
        <div class="text-h6">{{ data.name }} 剧集列表</div>
      </q-card-section>

      <q-tabs v-model="tab" dense active-color="primary" indicator-color="primary" align="justify" narrow-indicator>
        <q-tab
          v-for="item in categoryVideos"
          :key="item.season"
          :name="item.season"
          :label="`第 ${item.season} 季`"
          style="max-width: 150px"
        />
      </q-tabs>

      <q-separator />

      <q-card-section style="max-height: 40vh; overflow: auto">
        <div class="row items-center q-ml-md q-py-none">
          <q-checkbox
            :model-value="selectAllValue"
            indeterminate-value="maybe"
            @click="handleSelectAll"
            title="全选 / 反选"
          />

          <q-btn
            class="btn-download"
            color="primary"
            label="下载选中"
            flat
            :disable="selection.length === 0"
            @click="downloadSelection"
          />

          <q-btn
            class="btn-download"
            color="primary"
            icon="lock"
            title="锁定选中视频"
            flat
            :disable="selection.length === 0"
            @click="skipAll(true)"
          />

          <q-btn
            class="btn-download"
            color="primary"
            icon="lock_open"
            title="解锁选中视频"
            flat
            :disable="selection.length === 0"
            @click="skipAll(false)"
          />

          <q-space />

          <btn-dialog-search-subtitle
            search-package
            :package-episodes="currentTabEpisodes"
            label="搜索本季字幕包"
            size="md"
          />
          <btn-upload-multiple-for-tv :items="currentTabEpisodes" />
        </div>

        <q-tab-panels v-model="tab" animated>
          <q-tab-panel v-for="{ season, episodes } in categoryVideos" :key="season" :name="season" style="padding: 0">
            <q-list dense>
              <q-item v-for="item in episodes" :key="item.video_f_path">
                <q-item-section side top>
                  <q-checkbox v-model="selection" :val="item" />
                </q-item-section>

                <q-item-section>第 {{ padStart2(item.episode) }} 集</q-item-section>

                <q-item-section v-if="item.sub_f_path_list.length" side>
                  <btn-dialog-preview-video :subtitle-url-list="item.sub_url_list" :path="item.video_f_path" />
                </q-item-section>

                <q-item-section v-if="item.sub_f_path_list.length" side>
                  <q-btn color="primary" round flat dense icon="av_timer" title="校时间轴" @click.stop>
                    <q-popup-proxy anchor="top right">
                      <q-list dense class="tv-detail__subtitle-list">
                        <q-item
                          v-for="(subUrl, index) in item.sub_url_list"
                          :key="`${subUrl}-fix`"
                          clickable
                          v-ripple
                          v-close-popup
                          @click="
                            doFixSubtitleTimeline({
                              videoPath: item.video_f_path,
                              subPath: item.sub_f_path_list[index],
                            })
                          "
                        >
                          <q-item-section side>{{ index + 1 }}.</q-item-section>
                          <q-item-section class="overflow-hidden ellipsis" :title="subUrl.split(/\/|\\/).pop()">
                            {{ subUrl.split(/\/|\\/).pop() }}
                          </q-item-section>
                          <q-item-section side>
                            <q-icon name="av_timer" color="primary" size="18px" />
                          </q-item-section>
                        </q-item>
                      </q-list>
                    </q-popup-proxy>
                  </q-btn>
                </q-item-section>

                <q-item-section side>
                  <btn-upload-subtitle :path="item.video_f_path" />
                </q-item-section>

                <q-item-section side>
                  <btn-ignore-video :path="item.video_f_path" :video-type="VIDEO_TYPE_TV" />
                </q-item-section>

                <q-item-section side>
                  <q-btn
                    v-if="item.sub_f_path_list.length"
                    color="black"
                    round
                    flat
                    dense
                    icon="closed_caption"
                    @click.stop
                    title="已有字幕（点开后可手动校时间轴）"
                  >
                    <q-popup-proxy anchor="top right">
                      <q-list dense>
                        <q-item v-for="(subUrl, index) in item.sub_url_list" :key="subUrl">
                          <q-item-section side>{{ index + 1 }}.</q-item-section>

                          <q-item-section class="overflow-hidden ellipsis" :title="subUrl.split(/\/|\\/).pop()">
                            <a class="text-primary" :href="getUrl(subUrl)" target="_blank">
                              {{ subUrl.split(/\/|\\/).pop() }}
                            </a>
                          </q-item-section>
                        </q-item>
                      </q-list>
                    </q-popup-proxy>
                  </q-btn>
                  <q-btn v-else color="grey" round flat dense icon="closed_caption" @click.stop title="没有字幕" />
                </q-item-section>

                <q-item-section side>
                  <btn-dialog-search-subtitle
                    size="md"
                    round
                    :path="item.video_f_path"
                    :season="item.season"
                    :episode="item.episode"
                  />
                </q-item-section>

                <q-item-section side>
                  <q-btn
                    class="btn-download"
                    color="primary"
                    round
                    flat
                    dense
                    icon="download_for_offline"
                    title="加入下载队列"
                    @click="downloadSubtitle(item)"
                  />
                </q-item-section>
              </q-item>
            </q-list>
          </q-tab-panel>
        </q-tab-panels>
      </q-card-section>
    </q-card>
  </q-dialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue';
import LibraryApi from 'src/api/LibraryApi';
import { SystemMessage } from 'src/utils/message';
import { VIDEO_TYPE_TV } from 'src/constants/SettingConstants';
import { useQuasar } from 'quasar';
import { useSelection } from 'src/composables/use-selection';
import BtnIgnoreVideo from 'pages/library/BtnIgnoreVideo';
import eventBus from 'vue3-eventbus';
import BtnUploadSubtitle from 'pages/library/BtnUploadSubtitle';
import BtnDialogPreviewVideo from 'pages/library/BtnDialogPreviewVideo';
import BtnDialogSearchSubtitle from 'pages/library/BtnDialogSearchSubtitle';
import BtnUploadMultipleForTv from 'pages/library/tvs/BtnUploadMultipleForTv';
import { doFixSubtitleTimeline, getUrl } from 'pages/library/use-library';

const props = defineProps({
  data: Object,
});

const $q = useQuasar();
const visible = ref(false);
const tab = ref(null);

const categoryVideos = computed(() => {
  const result = [];
  props.data?.one_video_info?.forEach((item) => {
    const index = result.findIndex((entry) => entry.season === item.season);
    if (index === -1) {
      result.push({
        season: item.season,
        episodes: [item],
      });
      return;
    }
    result[index].episodes.push(item);
  });
  result.sort((a, b) => a.season - b.season);
  result.forEach((entry) => {
    entry.episodes.sort((a, b) => a.episode - b.episode);
  });
  return result;
});

watch(categoryVideos, () => {
  if (categoryVideos.value.length > 0 && tab.value === null) {
    tab.value = categoryVideos.value[0].season;
  }
});

const currentTabEpisodes = computed(
  () => categoryVideos.value.find((entry) => entry.season === tab.value)?.episodes ?? []
);

const { selectAllValue, handleSelectAll, selection } = useSelection(currentTabEpisodes);

watch(tab, () => {
  selection.value = [];
});

const padStart2 = (num) => {
  if (num < 10) {
    return `0${num}`;
  }
  return num;
};

const downloadSubtitle = async (items) => {
  const downloadList = items instanceof Array ? items : [items];
  $q.dialog({
    title: `加入 ${downloadList.length} 个下载任务`,
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
    const promises = downloadList.map(async (item) => {
      const [, err] = await LibraryApi.downloadSubtitle({
        video_type: VIDEO_TYPE_TV,
        physical_video_file_full_path: item.video_f_path,
        task_priority_level: val,
        media_server_inside_video_id: item.media_server_inside_video_id,
      });
      if (err !== null) {
        return Promise.reject(err);
      }
      return Promise.resolve();
    });

    const result = await Promise.allSettled(promises);
    const successCount = result.filter((item) => item.status === 'fulfilled').length;
    const errorCount = result.filter((item) => item.status === 'rejected').length;
    const msg = `成功加入 ${successCount} 个任务到下载队列${errorCount ? `，失败 ${errorCount} 个` : ''}`;
    SystemMessage.success(msg);
  });
};

const skipAll = async (isSkip) => {
  $q.dialog({
    title: `${isSkip ? '锁定' : '解锁'}选中视频`,
    message: `确定要${isSkip ? '锁定' : '解锁'}选中视频吗？`,
    cancel: true,
    persistent: true,
  }).onOk(async () => {
    const [, err] = await LibraryApi.setSkipInfo({
      video_skip_infos: selection.value.map((item) => ({
        video_type: VIDEO_TYPE_TV,
        physical_video_file_full_path: item.video_f_path,
        is_bluray: false,
        is_skip: isSkip,
      })),
    });
    if (err !== null) {
      SystemMessage.error(err.message);
      return;
    }

    const [res, err2] = await LibraryApi.getSkipInfo({
      video_skip_infos: selection.value.map((item) => ({
        video_type: VIDEO_TYPE_TV,
        physical_video_file_full_path: item.video_f_path,
        is_bluray: false,
        is_skip: true,
      })),
    });
    if (err2 !== null) {
      SystemMessage.error(err2.message);
      return;
    }

    selection.value.forEach((item, index) => {
      eventBus.emit(`refresh-skip-status-${item.video_f_path}`, res.is_skips[index]);
    });

    SystemMessage.success('操作成功');
  });
};

const downloadSelection = () => {
  downloadSubtitle(selection.value);
};
</script>

<style scoped lang="scss">
.tv-detail__subtitle-list {
  min-width: 300px;
}
</style>
