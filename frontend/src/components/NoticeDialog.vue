<template>
  <q-dialog v-model="visible" persistent>
    <q-card class="column notice-dialog-card">
      <q-card-section>
        <div class="text-h6 text-grey-8">当前仓库说明</div>
      </q-card-section>

      <q-separator />

      <q-card-section class="col notice-dialog-content">
        <markdown :source="notifyContent"></markdown>
      </q-card-section>

      <q-separator />

      <q-card-actions align="right" class="q-pa-md">
        <q-btn class="q-px-md" flat color="primary" label="我知道了" @click="handleAgree" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue';
import { LocalStorage } from 'quasar';
import { until } from '@vueuse/core';
import { systemState } from 'src/store/systemState';
import { normalizeDisplayVersion } from 'src/utils/version';
// eslint-disable-next-line import/no-webpack-loader-syntax
import notifyContent from 'raw-loader!../../NOTIFY.md';
import Markdown from 'components/Markdown';

const visible = ref(false);

const currentVersion = computed(() => normalizeDisplayVersion(systemState.systemInfo?.version));

const noticeFlagItemKey = computed(() => `noticeFlag-optimization-${currentVersion.value || 'unversioned'}`);

const handleAgree = () => {
  visible.value = false;
  LocalStorage.set(noticeFlagItemKey.value, true);
};

onMounted(async () => {
  await until(() => systemState.systemInfo !== null).toBe(true);
  const noticeFlag = LocalStorage.getItem(noticeFlagItemKey.value);
  if (!noticeFlag) {
    visible.value = true;
  }
});
</script>

<style scoped>
.notice-dialog-card {
  width: min(800px, 92vw);
  max-width: 92vw;
  max-height: 85vh;
}

.notice-dialog-content {
  overflow: auto;
}
</style>
