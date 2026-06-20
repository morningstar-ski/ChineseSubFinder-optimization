<template>
  <span @click="visible = true">
    <slot v-if="$slots.default"></slot>
    <q-badge v-else class="cursor-pointer" label="版本更新" title="查看当前版本更新" />
  </span>
  <q-dialog v-model="visible">
    <q-card class="column version-note-card">
      <q-card-section>
        <div class="text-h5">{{ dialogTitle }}</div>
      </q-card-section>

      <q-tabs
        v-model="tab"
        dense
        class="text-grey"
        active-color="primary"
        indicator-color="primary"
        align="justify"
        narrow-indicator
      >
        <q-tab name="summary" label="版本摘要" />
        <q-tab name="links" label="相关链接" />
      </q-tabs>

      <q-separator />

      <q-tab-panels v-model="tab" animated class="col">
        <q-tab-panel name="summary">
          <markdown :source="PROJECT_UPDATE_MARKDOWN" />
        </q-tab-panel>

        <q-tab-panel name="links" class="q-gutter-y-md">
          <section>
            <div class="text-h6">仓库入口</div>
            <div>
              代码和更新记录见
              <a :href="PROJECT_REPO_URL" target="_blank" rel="noreferrer">仓库主页</a>
            </div>
          </section>

          <section>
            <div class="text-h6">Docker</div>
            <div>
              部署文档见
              <a :href="PROJECT_DOCKER_DOC_URL" target="_blank" rel="noreferrer">Docker 文档</a>
            </div>
            <div class="text-grey">容器部署以本仓库文档为准。</div>
          </section>

          <section>
            <div class="text-h6">问题反馈</div>
            <div>
              问题反馈和变更跟踪见
              <a :href="PROJECT_ISSUES_URL" target="_blank" rel="noreferrer">问题列表</a>
            </div>
          </section>
        </q-tab-panel>
      </q-tab-panels>

      <q-separator />

      <q-card-actions align="right">
        <q-btn color="primary" @click="navigateToRepoPage">打开仓库</q-btn>
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup>
import { computed, ref } from 'vue';
import Markdown from 'components/Markdown';
import { systemState } from 'src/store/systemState';
import { normalizeDisplayVersion } from 'src/utils/version';
import {
  PROJECT_DOCKER_DOC_URL,
  PROJECT_ISSUES_URL,
  PROJECT_REPO_URL,
  PROJECT_UPDATE_MARKDOWN,
} from 'src/constants/ProjectLinks';

const visible = ref(false);
const tab = ref('summary');
const displayVersion = computed(() => normalizeDisplayVersion(systemState.systemInfo?.version));
const dialogTitle = computed(() => (displayVersion.value ? `当前版本更新 (${displayVersion.value})` : '当前版本更新'));

const navigateToRepoPage = () => {
  window.open(PROJECT_REPO_URL, '_blank');
  visible.value = false;
};
</script>

<style lang="scss" scoped>
.version-note-card {
  width: min(600px, calc(100vw - 32px));
  min-height: 400px;
}

a {
  color: $primary;
}
</style>
