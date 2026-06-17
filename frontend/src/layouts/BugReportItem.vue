<template>
  <q-item clickable class="app-toolbar-bug" @click="gotoGithubIssuePage" title="跳转到 Github issue 页面">
    <q-item-section>
      <div class="row items-center">
        <span><q-icon name="bug_report" size="18px" /></span>
        <span class="q-ml-xs">问题反馈</span>
      </div>
    </q-item-section>
  </q-item>
</template>

<script setup>
import { gotoGithubIssuePage } from 'src/utils/common';
</script>

<style scoped lang="scss">
.app-toolbar-bug {
  min-height: 40px;
  border-radius: 14px;
  color: #5b6c83;
}

.app-toolbar-bug:hover {
  background: #f3f6fb;
  color: #142033;
}
</style>
