<template>
  <div>
    <q-list class="settings-panel-list" dense>
      <q-item tag="label" v-ripple>
        <q-item-section>
          <q-item-label>接口异常通知</q-item-label>
          <q-item-label caption>仅维护者使用，普通部署无需配置。</q-item-label>
          <div v-if="form.enable" class="q-mt-sm">
            <q-input v-model="form.bark_server_address" standout dense />
          </div>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.enable" />
        </q-item-section>
      </q-item>
    </q-list>

    <q-separator class="q-mt-md" />
  </div>
</template>

<script setup>
import { formModel } from 'pages/settings/use-settings';
import { toRefs } from '@vueuse/core';

const { developer_settings: form } = toRefs(formModel);
</script>
