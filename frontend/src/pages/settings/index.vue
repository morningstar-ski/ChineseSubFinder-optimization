<template>
  <q-page class="page-shell settings-page">
    <div class="section-stack">
      <q-banner inline-actions class="settings-lock-banner app-surface-soft" v-if="isJobRunning">
        <template #avatar>
          <q-icon name="warning" color="warning" />
        </template>
        任务运行中，配置页当前为只读状态
        <template #action>
          <q-btn color="primary" label="去总览停止任务" flat @click="$router.push('/overview')" />
        </template>
      </q-banner>

      <section v-if="isSettingsLoaded" class="settings-shell section-stack">
        <q-card flat :class="['settings-shell__card', 'app-surface', { 'settings-shell__card--locked': isJobRunning }]">
          <q-tabs
            v-model="tab"
            dense
            active-color="primary"
            indicator-color="transparent"
            align="justify"
            narrow-indicator
            class="settings-tabs"
          >
            <q-tab name="basic" label="基础配置" class="settings-tabs__item" />
            <q-tab name="advanced" label="进阶配置" class="settings-tabs__item" />
            <q-tab name="subSource" label="字幕源设置" class="settings-tabs__item" />
            <q-tab name="emby" label="Emby 配置" class="settings-tabs__item" />
            <q-tab name="development" label="开发设置" class="settings-tabs__item" />
            <q-tab name="experiment" label="实验室" class="settings-tabs__item" />
          </q-tabs>

          <q-form @submit="submitAll" class="settings-shell__form">
            <q-tab-panels
              v-model="tab"
              animated
              class="settings-panels"
              :class="{ disabled: isJobRunning }"
              :style="{ pointerEvents: isJobRunning ? 'none' : 'default' }"
            >
              <q-tab-panel name="basic">
                <basic-settings />
              </q-tab-panel>

              <q-tab-panel name="advanced">
                <advanced-settings />
              </q-tab-panel>

              <q-tab-panel name="subSource">
                <sub-source-settings />
              </q-tab-panel>

              <q-tab-panel name="emby">
                <emby-settings />
              </q-tab-panel>

              <q-tab-panel name="development">
                <development-settings />
              </q-tab-panel>

              <q-tab-panel name="experiment">
                <experiment-settings />
              </q-tab-panel>
            </q-tab-panels>

            <q-separator class="settings-shell__separator" />

            <form-submit-area />
          </q-form>
        </q-card>
      </section>
    </div>
  </q-page>
</template>

<script setup>
import { computed, ref } from 'vue';
import BasicSettings from 'pages/settings/SettingsPanelBasic';
import AdvancedSettings from 'pages/settings/SettingsPanelAdvanced';
import EmbySettings from 'pages/settings/SettingsPanelEmby';
import DevelopmentSettings from 'pages/settings/SettingsPanelDevelopment';
import { formModel, submitAll, useSettings } from 'pages/settings/use-settings';
import { isJobRunning } from 'src/store/systemState';
import ExperimentSettings from 'pages/settings/SettingsPanelExperiment';
import FormSubmitArea from 'pages/settings/FormSubmitArea';
import SubSourceSettings from 'pages/settings/SettingsPanelSubSource';

const tab = ref('subSource');

const isSettingsLoaded = computed(() => Object.keys(formModel).length);

useSettings();
</script>

<style scoped lang="scss">
.settings-lock-banner {
  width: min(100%, 1120px);
  padding: 16px 18px;
  background: rgba(255, 255, 255, 0.92);
}

.settings-lock-banner :deep(.q-banner__content) {
  font-weight: 600;
  color: #223146;
}

.settings-lock-banner :deep(.q-banner__actions) {
  align-items: center;
}

.settings-shell {
  align-items: start;
}

.settings-shell__card {
  width: min(100%, 1120px);
  padding: 16px;
}

.settings-shell__card--locked {
  box-shadow: 0 20px 52px rgba(15, 23, 42, 0.06);
}

.settings-tabs {
  gap: 8px;
  padding: 6px;
  border-radius: 22px;
  background: #f4f7fb;
}

.settings-tabs :deep(.q-tab) {
  border-radius: 16px;
  color: #5d6e86;
}

.settings-tabs :deep(.q-tab--active) {
  background: #ffffff;
  color: #1677ff;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.06);
}

.settings-shell__form {
  display: grid;
  gap: 12px;
  width: min(100%, 1040px);
  margin-top: 16px;
}

.settings-panels {
  position: relative;
  border-radius: 22px;
  background: rgba(248, 250, 253, 0.86);
}

.settings-panels.disabled {
  opacity: 1 !important;
}

.settings-panels.disabled :deep(.disabled),
.settings-panels.disabled :deep([disabled]) {
  opacity: 1 !important;
}

.settings-panels.disabled :deep(.q-field--disabled .q-field__control),
.settings-panels.disabled :deep(.q-field__control[aria-disabled='true']) {
  background: rgba(250, 251, 253, 0.96) !important;
  box-shadow: inset 0 0 0 1px rgba(15, 23, 42, 0.04) !important;
}

.settings-panels.disabled :deep(.q-field__native),
.settings-panels.disabled :deep(.q-field__input),
.settings-panels.disabled :deep(.q-field__label),
.settings-panels.disabled :deep(.q-item__label),
.settings-panels.disabled :deep(.q-tab__label) {
  color: #7a8799 !important;
}

.settings-panels :deep(.q-tab-panel) {
  padding: 18px;
}

.settings-panels :deep(.settings-panel-list) {
  display: grid;
  gap: 12px;
  max-width: none !important;
}

.settings-panels :deep(.settings-panel-list > .q-item) {
  margin-bottom: 0;
  padding: 14px 16px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.82);
  border: 1px solid rgba(15, 23, 42, 0.05);
}

.settings-panels :deep(.settings-panel-list > .q-item > .q-item__section--main) {
  min-width: 0;
}

.settings-panels :deep(.settings-panel-list > .q-item > .q-item__section--avatar),
.settings-panels :deep(.settings-panel-list > .q-item > .q-item__section--side) {
  justify-content: flex-start;
  min-width: 0;
}

.settings-panels :deep(.settings-panel-list > .q-item > .q-item__section--avatar .q-field),
.settings-panels :deep(.settings-panel-list > .q-item > .q-item__section--avatar .q-select),
.settings-panels :deep(.settings-panel-list > .q-item > .q-item__section--side .q-field),
.settings-panels :deep(.settings-panel-list > .q-item > .q-item__section--side .q-select) {
  width: min(100%, 340px);
}

.settings-panels :deep(.settings-panel-list > .q-item > .q-item__section--avatar .row),
.settings-panels :deep(.settings-panel-list > .q-item > .q-item__section--side .row),
.settings-panels :deep(.settings-panel-list > .q-item > .q-item__section--avatar .column),
.settings-panels :deep(.settings-panel-list > .q-item > .q-item__section--side .column) {
  width: 100%;
}

.settings-panels :deep(.settings-panel-list .q-item .q-item) {
  margin-bottom: 8px;
  background: rgba(244, 247, 251, 0.86);
  border: 1px solid rgba(15, 23, 42, 0.04);
}

.settings-panels :deep(.q-item__label) {
  line-height: 1.6;
}

.settings-shell__separator {
  margin-top: 8px;
}

@media (max-width: 599px) {
  .settings-shell__card {
    padding: 16px;
  }

  .settings-lock-banner {
    padding: 14px 16px;
  }

  .settings-tabs {
    gap: 6px;
  }

  .settings-panels :deep(.q-tab-panel) {
    padding: 14px;
  }

  .settings-panels :deep(.settings-panel-list > .q-item) {
    padding: 12px 14px;
  }
}

@media (min-width: 1200px) {
  .settings-shell__form {
    width: min(76%, 1040px);
  }
}
</style>
