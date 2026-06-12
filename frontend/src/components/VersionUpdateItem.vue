<template>
  <span @click="visible = true">
    <slot v-if="$slots.default"></slot>
    <q-badge v-else class="cursor-pointer" label="说明" title="查看当前优化版说明" />
  </span>
  <q-dialog v-model="visible">
    <q-card class="column version-dialog-card">
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
        <q-tab name="summary" label="当前说明" />
        <q-tab name="update" label="获取更新" />
      </q-tabs>

      <q-separator />

      <q-tab-panels class="col version-dialog-content" v-model="tab" animated>
        <q-tab-panel name="summary">
          <markdown :source="PROJECT_UPDATE_MARKDOWN" />
        </q-tab-panel>
        <q-tab-panel name="update">
          <section>
            <div class="text-h6">仓库入口</div>
            <div>
              当前优化版代码、说明和提交记录都在
              <a :href="PROJECT_REPO_URL" target="_blank">仓库主页</a>
            </div>
          </section>

          <section class="q-mt-md">
            <div class="text-h6">Docker</div>
            <div>
              参考
              <a :href="PROJECT_DOCKER_DOC_URL" target="_blank">Docker 部署文档</a>
            </div>
            <div class="text-grey">* 帮助文档和部署说明已经统一指向当前优化版仓库。</div>
          </section>

          <section class="q-mt-md">
            <div class="text-h6">问题反馈</div>
            <div>
              使用本仓库的 issue 跟踪问题，
              <a :href="PROJECT_ISSUES_URL" target="_blank">问题列表</a>
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

const dialogTitle = computed(() => (displayVersion.value ? `优化版说明 (${displayVersion.value})` : '优化版说明'));

const navigateToRepoPage = () => {
  window.open(PROJECT_REPO_URL, '_blank');
  visible.value = false;
};
</script>

<style lang="scss" scoped>
a {
  color: $primary;
}

.version-dialog-card {
  width: min(600px, 92vw);
  max-width: 92vw;
  min-height: min(400px, 85vh);
  max-height: 85vh;
}

.version-dialog-content {
  overflow: auto;
}
</style>
