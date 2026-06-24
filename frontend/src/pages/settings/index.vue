<template>
  <q-page class="page-shell settings-page">
    <div class="section-stack">
      <q-banner v-if="isJobRunning" inline-actions class="settings-lock-banner app-surface-soft">
        <template #avatar>
          <q-icon name="warning" color="warning" />
        </template>
        任务运行中，当前配置页为只读状态。
        <template #action>
          <q-btn color="primary" label="前往总览停止任务" flat @click="$router.push('/overview')" />
        </template>
      </q-banner>

      <section v-if="isSettingsLoaded" class="settings-shell section-stack">
        <q-card flat :class="['settings-shell__card', 'app-surface', { 'settings-shell__card--locked': isJobRunning }]">
          <q-tabs
            v-model="tab"
            dense
            active-color="primary"
            indicator-color="transparent"
            align="left"
            narrow-indicator
            class="settings-tabs"
          >
            <q-tab name="library" label="影视库" class="settings-tabs__item" />
            <q-tab name="providers" label="字幕策略" class="settings-tabs__item" />
            <q-tab name="runtime" label="下载规则" class="settings-tabs__item" />
            <q-tab name="system" label="扩展与维护" class="settings-tabs__item" />
          </q-tabs>

          <q-form @submit="submitAll" class="settings-shell__form">
            <q-tab-panels
              v-model="tab"
              animated
              class="settings-panels"
              :class="{ disabled: isJobRunning }"
              :style="{ pointerEvents: isJobRunning ? 'none' : 'default' }"
            >
              <q-tab-panel name="library">
                <div class="settings-section-stack">
                  <section class="settings-section">
                    <div class="settings-section__header">
                      <h2 class="settings-section__title">影视库路径</h2>
                      <p class="settings-section__subtitle">管理电影、剧集目录，以及扫描周期和基础并发设置。</p>
                    </div>
                    <div class="settings-section__body">
                      <basic-settings />
                    </div>
                  </section>

                  <section class="settings-section">
                    <div class="settings-section__header">
                      <h2 class="settings-section__title">Emby 同步</h2>
                      <p class="settings-section__subtitle">集中管理 Emby 连接、同步方式和媒体服务器联动行为。</p>
                    </div>
                    <div class="settings-section__body">
                      <emby-settings />
                    </div>
                  </section>
                </div>
              </q-tab-panel>

              <q-tab-panel name="providers">
                <div class="settings-section-stack">
                  <section class="settings-section">
                    <div class="settings-section__header">
                      <h2 class="settings-section__title">字幕源与回退</h2>
                      <p class="settings-section__subtitle">统一管理字幕供应商、下载顺序和显式回退开关。</p>
                    </div>
                    <div class="settings-section__body">
                      <sub-source-settings />
                    </div>
                  </section>
                </div>
              </q-tab-panel>

              <q-tab-panel name="runtime">
                <div class="settings-section-stack">
                  <section class="settings-section">
                    <div class="settings-section__header">
                      <h2 class="settings-section__title">下载规则与网络</h2>
                      <p class="settings-section__subtitle">统一管理代理、任务队列、字幕命名、时间轴和 TMDB 配置。</p>
                    </div>
                    <div class="settings-section__body">
                      <advanced-settings />
                    </div>
                  </section>
                </div>
              </q-tab-panel>

              <q-tab-panel name="system">
                <div class="settings-section-stack">
                  <section class="settings-section">
                    <div class="settings-section__header">
                      <h2 class="settings-section__title">扩展能力</h2>
                      <p class="settings-section__subtitle">集中放置浏览器、LLM 回退、编码转换和开放接口等扩展项。</p>
                    </div>
                    <div class="settings-section__body">
                      <experiment-settings />
                    </div>
                  </section>

                  <section class="settings-section">
                    <div class="settings-section__header">
                      <h2 class="settings-section__title">通知与维护</h2>
                      <p class="settings-section__subtitle">保留异常通知和维护相关选项，避免和核心配置混在一起。</p>
                    </div>
                    <div class="settings-section__body">
                      <development-settings />
                    </div>
                  </section>
                </div>
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

const tab = ref('library');

const isSettingsLoaded = computed(() => Object.keys(formModel).length);

useSettings();
</script>

<style scoped lang="scss">
.settings-lock-banner {
  box-sizing: border-box;
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
  min-width: 0;
}

.settings-shell__card {
  box-sizing: border-box;
  width: min(100%, 1100px);
  min-width: 0;
  padding: 18px 20px 20px;
}

.settings-shell__card--locked {
  box-shadow: 0 20px 52px rgba(15, 23, 42, 0.06);
}

.settings-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
  padding: 6px;
  border-radius: 22px;
  background: #f4f7fb;
}

.settings-tabs :deep(.q-tab) {
  min-width: 0;
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
  width: min(100%, 840px);
  min-width: 0;
  margin-top: 16px;
}

.settings-panels {
  position: relative;
  min-width: 0;
  border-radius: 22px;
  background: transparent;
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

.settings-panels :deep(.q-field__control) {
  background: #ffffff !important;
  color: #142033 !important;
  box-shadow: inset 0 0 0 1px rgba(15, 23, 42, 0.08), 0 8px 22px rgba(15, 23, 42, 0.05) !important;
}

.settings-panels :deep(.q-field__native),
.settings-panels :deep(.q-field__input),
.settings-panels :deep(.q-field__label),
.settings-panels :deep(.q-icon) {
  color: #142033 !important;
}

.settings-panels :deep(.q-field__native::placeholder),
.settings-panels :deep(.q-field__input::placeholder) {
  color: #7f8ea3 !important;
  opacity: 1;
}

.settings-panels :deep(.q-tab-panel) {
  padding: 0;
}

.settings-section-stack {
  display: grid;
  gap: 16px;
  min-width: 0;
}

.settings-section {
  min-width: 0;
  overflow: hidden;
  border: 1px solid rgba(15, 23, 42, 0.05);
  border-radius: 22px;
  background: rgba(248, 250, 253, 0.86);
}

.settings-section__header {
  padding: 18px 20px 12px;
  border-bottom: 1px solid rgba(15, 23, 42, 0.05);
  background: rgba(255, 255, 255, 0.68);
}

.settings-section__title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  line-height: 1.3;
  color: #142033;
}

.settings-section__subtitle {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 1.55;
  color: #66768f;
}

.settings-section__body {
  padding: 16px;
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
  width: min(100%, 280px);
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
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 6px;
  }

  .settings-tabs :deep(.q-tabs__content) {
    flex-wrap: wrap;
    min-width: 0;
  }

  .settings-tabs :deep(.q-tab) {
    width: 100%;
    min-height: 40px;
    padding: 0 10px;
  }

  .settings-tabs :deep(.q-tab__label) {
    font-size: 13px;
  }

  .settings-section__header {
    padding: 14px 16px 10px;
  }

  .settings-section__body {
    padding: 12px;
  }

  .settings-panels :deep(.settings-panel-list > .q-item) {
    padding: 12px 14px;
  }
}

@media (min-width: 1200px) {
  .settings-shell__form {
    width: min(74%, 840px);
  }
}
</style>
