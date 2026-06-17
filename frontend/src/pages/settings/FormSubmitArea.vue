<template>
  <div class="submit-bar">
    <div class="submit-bar__group">
      <q-btn :disable="isJobRunning" label="保存" color="primary" type="submit" :loading="submitting" icon="save" />
      <q-btn :disable="isJobRunning" label="重置" outline color="primary" @click="resetForm" icon="restart_alt" />
    </div>

    <div class="submit-bar__group submit-bar__group--right">
      <export-setting-btn-dialog />
    </div>
  </div>
</template>

<script setup>
import { resetForm, submitting } from 'pages/settings/use-settings';
import ExportSettingBtnDialog from 'pages/settings/BtnDialogExportSetting';
import { isJobRunning } from 'src/store/systemState';
</script>

<style scoped lang="scss">
.submit-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 4px 4px;
  border-top: 1px solid rgba(15, 23, 42, 0.06);
}

.submit-bar__group {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.submit-bar__group--right {
  margin-left: auto;
}
</style>
