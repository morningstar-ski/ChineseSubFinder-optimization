<template>
  <q-page class="q-pa-md">
    <q-banner inline-actions class="text-white bg-red" v-if="isJobRunning">
      <template v-slot:avatar>
        <q-icon name="warning" />
      </template>
      任务进程运行中，不能更改配置
      <template v-slot:action>
        <q-btn color="white" label="去总览页面停止" flat @click="$router.push('/overview')" />
      </template>
      <span> </span>
    </q-banner>
    <q-banner inline-actions class="bg-blue-1 text-blue-10 q-mb-md">
      <template v-slot:avatar>
        <q-icon name="info" />
      </template>
      未注入版本号时不再显示占位版本；进阶配置里的“自动校正字幕时间轴”当前仍是内置 ffsubsync 复刻方案；远程 Chrome
      现在只接了 ws 地址和可选用户目录，当前主要影响浏览器型字幕源。
      <template v-slot:action>
        <q-btn flat color="primary" label="仓库" @click="openPage(PROJECT_REPO_URL)" />
        <q-btn flat color="primary" label="Issues" @click="openPage(PROJECT_ISSUES_URL)" />
      </template>
    </q-banner>
    <q-card v-if="isSettingsLoaded" flat>
      <q-tabs
        v-model="tab"
        dense
        active-color="primary"
        indicator-color="primary"
        align="justify"
        narrow-indicator
        style="max-width: 700px"
      >
        <q-tab name="basic" label="基础配置" />
        <q-tab name="advanced" label="进阶配置" />
        <q-tab name="subSource" label="字幕源设置" />
        <q-tab name="emby" label="Emby配置" />
        <q-tab name="development" label="开发人员配置" />
        <q-tab name="experiment" label="实验室" />
      </q-tabs>

      <q-separator />

      <q-form @submit="submitAll">
        <q-tab-panels
          v-model="tab"
          animated
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

        <q-separator />

        <form-submit-area />
      </q-form>
    </q-card>
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
import { PROJECT_ISSUES_URL, PROJECT_REPO_URL } from 'src/constants/ProjectLinks';

const tab = ref('subSource');

const isSettingsLoaded = computed(() => Object.keys(formModel).length);

const openPage = (url) => {
  window.open(url, '_blank');
};

useSettings();
</script>
