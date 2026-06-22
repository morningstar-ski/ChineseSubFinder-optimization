<template>
  <div>
    <q-list class="settings-panel-list" dense>
      <q-item class="basic-settings__scan-item">
        <q-item-section>
          <q-item-label>字幕扫描时机</q-item-label>
          <div class="scan-mode-list">
            <div class="scan-mode-row">
              <div class="scan-mode-row__radio">
                <q-radio v-model="scanType" :val="0" />
              </div>
              <div class="scan-mode-row__content">
                <div class="scan-mode-row__title">扫描的间隔</div>
                <div class="scan-mode-row__caption">间隔小时数</div>
              </div>
              <div class="scan-mode-row__control">
                <q-select
                  v-model="scanCronString0"
                  :options="scanIntervalOptions"
                  standout
                  dense
                  class="scan-mode-row__field"
                  :rules="[(val) => !!val || '不能为空']"
                  emit-value
                  map-options
                  :disable="scanType !== 0"
                  @update:model-value="handleScanIntervalChange"
                />
              </div>
            </div>

            <div class="scan-mode-row">
              <div class="scan-mode-row__radio">
                <q-radio v-model="scanType" :val="1" />
              </div>
              <div class="scan-mode-row__content">
                <div class="scan-mode-row__title">指定扫描时间</div>
                <div class="scan-mode-row__caption">选择每天固定时间点</div>
              </div>
              <div class="scan-mode-row__control">
                <q-select
                  v-model="scanCronString1"
                  :options="scanSpecTimeOptions"
                  standout
                  dense
                  class="scan-mode-row__field"
                  :rules="[
                    (val) => !!val || !!val?.length || '不能为空',
                    (val) => val.length <= 4 || '最多选择4个时间点',
                  ]"
                  emit-value
                  map-options
                  :disable="scanType !== 1"
                  @update:model-value="handleScanSpecTimeChange"
                  multiple
                />
              </div>
            </div>

            <div class="scan-mode-row">
              <div class="scan-mode-row__radio">
                <q-radio v-model="scanType" :val="2" />
              </div>
              <div class="scan-mode-row__content">
                <div class="scan-mode-row__title">自定义规则</div>
                <div class="scan-mode-row__caption">
                  详细规则参考
                  <a href="https://pkg.go.dev/github.com/robfig/cron/v3" target="_blank" class="text-primary"
                    >robfig/cron 文档</a
                  >
                </div>
              </div>
              <div class="scan-mode-row__control">
                <q-input
                  v-model="scanCronString2"
                  standout
                  dense
                  class="scan-mode-row__field"
                  :rules="[(val) => !!val || '不能为空', validateCronTime]"
                  @update:model-value="handleScanCustomChange"
                  :disable="scanType !== 2"
                />
              </div>
            </div>

            <div class="scan-mode-row scan-mode-row--compact">
              <div class="scan-mode-row__radio">
                <q-radio v-model="scanType" :val="3" @update:model-value="handleScanNoScanChange" />
              </div>
              <div class="scan-mode-row__content">
                <div class="scan-mode-row__title">不扫描</div>
              </div>
            </div>
          </div>
        </q-item-section>
      </q-item>

      <q-separator spaced inset />

      <q-item class="basic-settings__performance-item">
        <q-item-section class="basic-settings__performance-label">
          <q-item-label>运行性能档位</q-item-label>
        </q-item-section>
        <q-item-section class="basic-settings__performance-options">
          <div class="basic-settings__performance-group">
            <q-radio v-model="form.threads" :val="1" label="低占用（1 线程）" />
            <q-radio v-model="form.threads" :val="3" label="标准（3 线程）" />
            <q-radio v-model="form.threads" :val="6" label="高性能（6 线程）" />
          </div>
        </q-item-section>
      </q-item>

      <q-separator spaced inset />

      <q-item class="basic-settings__path-item">
        <q-item-section class="items-start basic-settings__path-label" top>
          <q-item-label>电影目录</q-item-label>
        </q-item-section>
        <q-item-section class="basic-settings__path-editor-section">
          <div v-if="!form.movie_paths?.length" class="path-editor path-editor--empty">
            <q-btn
              icon="add"
              color="primary"
              dense
              rounded
              size="xs"
              title="新增"
              @click="form.movie_paths.push('')"
            ></q-btn>
          </div>
          <div v-else class="path-editor">
            <div v-for="(item, i) in form.movie_paths" :key="`movie-${i}`" class="path-editor__row">
              <q-input
                v-model="form.movie_paths[i]"
                class="path-editor__input"
                placeholder="/media/电影"
                standout
                dense
                lazy-rules
                :rules="[(val) => !!val || '不能为空', validateRemotePath]"
              />
              <q-btn
                v-if="i === 0"
                icon="add"
                color="primary"
                dense
                rounded
                size="xs"
                title="新增"
                @click="form.movie_paths.push('')"
              ></q-btn>
              <q-btn
                v-else
                icon="remove"
                color="negative"
                dense
                rounded
                size="xs"
                title="删除"
                @click="form.movie_paths.splice(i, 1)"
              ></q-btn>
            </div>
          </div>
        </q-item-section>
      </q-item>

      <q-separator spaced inset />

      <q-item class="basic-settings__path-item">
        <q-item-section class="items-start basic-settings__path-label" top>
          <q-item-label>剧集目录</q-item-label>
        </q-item-section>
        <q-item-section class="basic-settings__path-editor-section">
          <div v-if="!form.series_paths?.length" class="path-editor path-editor--empty">
            <q-btn
              icon="add"
              color="primary"
              dense
              rounded
              size="xs"
              title="新增"
              @click="form.series_paths.push('')"
            ></q-btn>
          </div>
          <div v-else class="path-editor">
            <div v-for="(item, i) in form.series_paths" :key="`series-${i}`" class="path-editor__row">
              <q-input
                v-model="form.series_paths[i]"
                class="path-editor__input"
                placeholder="/media/连续剧"
                standout
                dense
                :rules="[(val) => !!val || '不能为空', validateRemotePath]"
              />
              <q-btn
                v-if="i === 0"
                icon="add"
                color="primary"
                dense
                rounded
                size="xs"
                title="新增"
                @click="form.series_paths.push('')"
              ></q-btn>
              <q-btn
                v-else
                icon="remove"
                color="negative"
                dense
                rounded
                size="xs"
                title="删除"
                @click="form.series_paths.splice(i, 1)"
              ></q-btn>
            </div>
          </div>
        </q-item-section>
      </q-item>
    </q-list>
  </div>
</template>

<script setup>
import { formModel } from 'pages/settings/use-settings';
import { validateCronTime, validateRemotePath } from 'src/utils/quasar-validators';
import { toRefs } from '@vueuse/core';
import { ref, watch } from 'vue';

const { common_settings: form } = toRefs(formModel);

const NO_SCAN_CRON_RULE = '@every 87600h';

const scanCronString0 = ref('');
const scanCronString1 = ref([]);
const scanCronString2 = ref('');
const scanType = ref(0);

if (form.value.scan_interval === NO_SCAN_CRON_RULE) {
  scanType.value = 3;
} else if (form.value.interval_or_assign_or_custom === 0) {
  scanType.value = 0;
  scanCronString0.value = form.value.scan_interval.split(' ').pop();
} else if (form.value.interval_or_assign_or_custom === 1) {
  scanType.value = 1;
  scanCronString1.value = form.value.scan_interval.split(' ')[1].split(',');
} else if (form.value.interval_or_assign_or_custom === 2) {
  scanType.value = 2;
  scanCronString2.value = form.value.scan_interval;
}

const scanIntervalOptions = [
  { label: '每4小时', value: '4h' },
  { label: '每5小时', value: '5h' },
  { label: '每6小时', value: '6h' },
  { label: '每7小时', value: '7h' },
  { label: '每8小时', value: '8h' },
  { label: '每9小时', value: '9h' },
  { label: '每10小时', value: '10h' },
];
const scanSpecTimeOptions = [
  { label: '00:00', value: '0' },
  { label: '01:00', value: '1' },
  { label: '02:00', value: '2' },
  { label: '03:00', value: '3' },
  { label: '04:00', value: '4' },
  { label: '05:00', value: '5' },
  { label: '06:00', value: '6' },
  { label: '07:00', value: '7' },
  { label: '08:00', value: '8' },
  { label: '09:00', value: '9' },
  { label: '10:00', value: '10' },
  { label: '11:00', value: '11' },
  { label: '12:00', value: '12' },
  { label: '13:00', value: '13' },
  { label: '14:00', value: '14' },
  { label: '15:00', value: '15' },
  { label: '16:00', value: '16' },
  { label: '17:00', value: '17' },
  { label: '18:00', value: '18' },
  { label: '19:00', value: '19' },
  { label: '20:00', value: '20' },
  { label: '21:00', value: '21' },
  { label: '22:00', value: '22' },
  { label: '23:00', value: '23' },
];

const handleScanIntervalChange = () => {
  formModel.common_settings.interval_or_assign_or_custom = 0;
  formModel.common_settings.scan_interval = `@every ${scanCronString0.value}`;
};

const handleScanSpecTimeChange = () => {
  formModel.common_settings.interval_or_assign_or_custom = 1;
  formModel.common_settings.scan_interval = `0 ${scanCronString1.value.join(',')} * * *`;
};

const handleScanCustomChange = () => {
  formModel.common_settings.interval_or_assign_or_custom = 2;
  formModel.common_settings.scan_interval = `${scanCronString2.value}`;
};

const handleScanNoScanChange = () => {
  formModel.common_settings.interval_or_assign_or_custom = 2;
  formModel.common_settings.scan_interval = NO_SCAN_CRON_RULE;
};

// 同步更新emby的线程设置
watch(
  () => formModel.common_settings.threads,
  (val) => {
    formModel.emby_settings.threads = val;
  }
);
</script>

<style scoped lang="scss">
.basic-settings__performance-item,
.basic-settings__path-item {
  gap: 16px;
}

.basic-settings__performance-label,
.basic-settings__path-label {
  flex: 0 0 132px;
}

.basic-settings__performance-options,
.basic-settings__path-editor-section {
  min-width: 0;
  flex: 1 1 auto;
}

.basic-settings__performance-group {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 20px;
}

.scan-mode-list {
  display: grid;
  gap: 8px;
  margin-top: 8px;
}

.scan-mode-row {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr) minmax(160px, 200px);
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  border: 1px solid rgba(15, 23, 42, 0.04);
  border-radius: 18px;
  background: rgba(244, 247, 251, 0.86);
}

.scan-mode-row--compact {
  grid-template-columns: 28px minmax(0, 1fr);
}

.scan-mode-row__radio {
  display: flex;
  align-items: flex-start;
  justify-content: center;
}

.scan-mode-row__content {
  min-width: 0;
}

.scan-mode-row__title {
  font-size: 15px;
  font-weight: 600;
  line-height: 1.4;
  color: #142033;
}

.scan-mode-row__caption {
  margin-top: 4px;
  font-size: 13px;
  line-height: 1.5;
  color: #66768f;
}

.scan-mode-row__field {
  width: 100%;
}

.path-editor {
  display: grid;
  gap: 12px;
  width: min(100%, 520px);
}

.path-editor--empty {
  justify-items: start;
}

.path-editor__row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
}

.path-editor__input {
  min-width: 0;
}

@media (max-width: 768px) {
  .scan-mode-row {
    grid-template-columns: 28px minmax(0, 1fr);
    gap: 10px 12px;
  }

  .scan-mode-row__control {
    grid-column: 2 / -1;
  }

  .scan-mode-row__title {
    font-size: 14px;
  }

  .basic-settings__performance-item,
  .basic-settings__path-item {
    flex-wrap: wrap;
    gap: 12px;
  }

  .basic-settings__performance-label,
  .basic-settings__path-label,
  .basic-settings__performance-options,
  .basic-settings__path-editor-section {
    flex-basis: 100%;
  }

  .basic-settings__performance-group {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .path-editor {
    width: 100%;
  }
}
</style>
