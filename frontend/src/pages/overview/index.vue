<template>
  <q-page class="page-shell overview-page">
    <div v-if="systemState.jobStatus" class="section-stack">
      <section class="overview-hero app-surface">
        <div class="overview-hero__copy">
          <h1 class="overview-hero__title">任务引擎</h1>
          <p class="overview-hero__desc">查看当前任务状态，并手动启动或停止任务。</p>

          <div class="overview-hero__actions">
            <q-btn
              v-if="isJobRunning"
              label="强制停止"
              color="negative"
              icon="stop_circle"
              @click="stopJobs"
              :loading="submitting"
            />
            <q-btn
              v-else
              label="立即运行"
              color="primary"
              icon="play_circle"
              @click="startJobs"
              :loading="submitting"
            />
          </div>
        </div>

        <div class="overview-hero__status">
          <div class="overview-status-card app-surface-soft">
            <div class="overview-status-card__label">当前状态</div>
            <div class="overview-status-card__value" :class="{ 'is-running': isJobRunning }">
              {{ isJobRunning ? '运行中' : '待命中' }}
            </div>
            <q-badge :color="isJobRunning ? 'positive' : 'grey-6'" class="overview-status-card__badge">
              {{ isJobRunning ? '任务运行中' : '等待启动' }}
            </q-badge>
          </div>
        </div>
      </section>

      <section class="metric-grid">
        <article v-for="item in statusCards" :key="item.label" class="metric-card app-surface">
          <div class="metric-card__label">{{ item.label }}</div>
          <div class="metric-card__value">{{ item.value }}</div>
          <div class="metric-card__hint">{{ item.hint }}</div>
        </article>
      </section>
    </div>
  </q-page>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue';
import { getJobsStatus, isJobRunning, systemState } from 'src/store/systemState';
import { useQuasar } from 'quasar';
import JobApi from 'src/api/JobApi';
import { SystemMessage } from 'src/utils/message';

const $q = useQuasar();

const submitting = ref(false);

const statusCards = computed(() => [
  {
    label: '写保护',
    value: isJobRunning.value ? '已锁定' : '可编辑',
    hint: isJobRunning.value ? '运行中会锁定部分设置面板，避免运行态改配置。' : '当前可以调整下载与回退配置。',
  },
  {
    label: '调度模式',
    value: isJobRunning.value ? '后台执行中' : '等待触发',
    hint: isJobRunning.value ? '后台正在执行当前批次任务。' : '可以从这里直接拉起一轮任务处理。',
  },
  {
    label: '操作入口',
    value: isJobRunning.value ? '停止' : '启动',
    hint: isJobRunning.value ? '如需改配置，先停掉当前任务。' : '适合做手动验证或临时跑一轮。',
  },
]);

const startJobs = () => {
  $q.dialog({
    title: '是否立即运行？',
    cancel: true,
  }).onOk(async () => {
    submitting.value = true;
    const [, err] = await JobApi.start();
    submitting.value = false;
    if (err !== null) {
      SystemMessage.error(err.message);
      return;
    }
    getJobsStatus();
    SystemMessage.success('启动成功');
  });
};

const stopJobs = () => {
  $q.dialog({
    title: '是否强制停止？',
    cancel: true,
  }).onOk(async () => {
    submitting.value = true;
    const [, err] = await JobApi.stop();
    submitting.value = false;
    if (err !== null) {
      SystemMessage.error(err.message);
      return;
    }
    getJobsStatus();
    SystemMessage.success('停止成功');
  });
};

onMounted(() => {
  getJobsStatus();
});
</script>

<style scoped lang="scss">
.overview-page {
  display: grid;
}

.overview-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.6fr) minmax(260px, 0.8fr);
  gap: 18px;
  padding: 26px;
}

.overview-hero__title {
  margin: 0;
  font-size: clamp(30px, 4vw, 42px);
  line-height: 1;
  font-weight: 700;
  color: #142033;
}

.overview-hero__desc {
  max-width: 560px;
  margin: 14px 0 0;
  font-size: 14px;
  line-height: 1.7;
  color: #61728a;
}

.overview-hero__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 22px;
}

.overview-hero__status {
  display: flex;
  align-items: stretch;
}

.overview-status-card {
  display: grid;
  align-content: space-between;
  width: 100%;
  min-height: 100%;
  padding: 20px;
}

.overview-status-card__label {
  font-size: 12px;
  font-weight: 700;
  color: #7f8ea3;
}

.overview-status-card__value {
  margin-top: 10px;
  font-size: 30px;
  font-weight: 700;
  line-height: 1;
  color: #142033;
}

.overview-status-card__value.is-running {
  color: #2bb673;
}

.overview-status-card__badge {
  margin-top: 18px;
  width: fit-content;
}

@media (max-width: 899px) {
  .overview-hero {
    grid-template-columns: 1fr;
  }
}
</style>
