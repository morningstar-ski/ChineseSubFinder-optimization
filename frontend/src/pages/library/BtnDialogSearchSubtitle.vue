<template>
  <q-btn color="primary" icon="search" size="sm" flat dense v-bind="$attrs" @click="visible = true" title="字幕搜索" />

  <q-dialog v-model="visible" transition-show="slide-up" transition-hide="slide-down" persistent>
    <q-card style="min-width: 70vw">
      <q-card-section>
        <div class="row justify-between items-center">
          <div class="text-h6 text-grey-8">字幕搜索</div>
          <q-btn icon="close" flat round dense @click="visible = false" />
        </div>
        <div class="text-warning">* 下载字幕包会在浏览器中处理，下载期间不要关闭当前页面</div>
      </q-card-section>
      <q-separator />

      <template v-if="!searchPackage">
        <search-panel-manual :is-movie="isMovie" :path="path" />
      </template>
      <template v-else>
        <q-card-section class="text-grey-7"
          >整季远端字幕包入口已移除，请按单集手动搜索或直接上传本地字幕。</q-card-section
        >
      </template>
    </q-card>
  </q-dialog>
</template>

<script setup>
import { ref } from 'vue';
import SearchPanelManual from 'pages/library/SearchPanelManual.vue';

defineProps({
  path: String,
  isMovie: {
    type: Boolean,
    default: false,
  },
  searchPackage: {
    type: Boolean,
    default: false,
  },
  season: {
    type: Number,
  },
  episode: {
    type: Number,
  },
  packageEpisodes: {
    type: Array,
  },
});

const visible = ref(false);
</script>
