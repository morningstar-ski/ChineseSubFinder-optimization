<template>
  <q-page class="page-shell movie-index">
    <div class="page-toolbar">
      <div class="page-toolbar__group">
        <btn-dialog-library-refresh />
        <btn-dialog-media-server-subtitle-refresh />
      </div>

      <div class="page-toolbar__spacer"></div>

      <q-input v-model="filterForm.search" outlined dense label="输入关键词搜索" class="movie-index__search">
        <template #append>
          <q-icon name="search" />
        </template>
      </q-input>
    </div>

    <q-separator class="q-mb-md" />

    <div v-if="movies.length" class="movie-index__grid">
      <q-intersection v-for="item in filteredMovies" once :key="item.video_f_path" style="height: 280px">
        <list-item-movie :data="item" width="180px" cover-height="220px" />
      </q-intersection>
    </div>

    <div v-else class="q-my-md text-grey">当前没有可用视频，点击“更新缓存”可以重新建立缓存。</div>
  </q-page>
</template>

<script setup>
import { useLibrary } from 'pages/library/use-library';
import { computed, reactive } from 'vue';
import BtnDialogLibraryRefresh from 'pages/library/BtnLibraryRefresh';
import BtnDialogMediaServerSubtitleRefresh from 'pages/library/BtnMediaServerSubtitleRefresh';
import ListItemMovie from './ListItemMovie';

const filterForm = reactive({
  hasSubtitle: null,
  search: '',
});

const { movies } = useLibrary();

const filteredMovies = computed(() => {
  let res = movies.value;

  if (filterForm.hasSubtitle === true) {
    res = movies.value.filter((item) => item.sub_f_path_list.length > 0);
  }
  if (filterForm.hasSubtitle === false) {
    res = movies.value.filter((item) => item.sub_f_path_list.length === 0);
  }

  if (filterForm.search !== '') {
    res = res.filter((item) => item.name.toLowerCase().includes(filterForm.search.toLowerCase()));
  }

  return res;
});
</script>

<style scoped lang="scss">
.movie-index__search {
  width: min(100%, 300px);
}

.movie-index__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 18px;
}
</style>
