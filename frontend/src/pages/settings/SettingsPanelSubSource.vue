<template>
  <div class="sub-source-panel">
    <section class="sub-source-section">
      <div class="sub-source-section__header">
        <h3 class="sub-source-section__title">字幕源地址与限额</h3>
        <p class="sub-source-section__subtitle">统一维护各字幕源的根地址和每日限额。</p>
      </div>

      <div class="sub-source-provider-list">
        <div v-for="item in visibleSuppliers" :key="item.name" class="sub-source-provider-row">
          <div class="sub-source-provider-row__meta">
            <div class="sub-source-provider-row__name">{{ item.name }}</div>
            <div class="sub-source-provider-row__url">{{ item.root_url }}</div>
            <div v-if="item.name !== 'csf'" class="sub-source-provider-row__limit">
              每日下载限制：{{ item.daily_download_limit }}
            </div>
          </div>
          <edit-sub-source-btn-dialog :data="item" @update="(data) => handleSubSourceUpdate(item, data)" />
        </div>
      </div>
    </section>

    <section class="sub-source-section">
      <div class="sub-source-section__header">
        <h3 class="sub-source-section__title">供应商配置</h3>
        <p class="sub-source-section__subtitle">每个源在同一块里完成启用和凭据填写。</p>
      </div>

      <div class="supplier-card-list">
        <article class="supplier-card">
          <div class="supplier-card__top">
            <div class="supplier-card__heading">
              <div class="supplier-card__title-row">
                <h4 class="supplier-card__title">Assrt</h4>
                <span class="supplier-card__link">https://assrt.net/api/doc</span>
              </div>
              <p class="supplier-card__caption">
                注册：https://assrt.net/user/register.xml，用户面板：https://assrt.net/usercp.php
              </p>
            </div>
            <q-toggle v-model="form.assrt_settings.enabled" color="primary" />
          </div>

          <ul class="supplier-card__tips">
            <li>普通用户接口频率较低，建议按需开启。</li>
            <li>保存后立即生效，无需重启程序或容器。</li>
            <li>不使用时直接关闭即可。</li>
          </ul>

          <q-input
            v-model="form.assrt_settings.token"
            :disable="!form.assrt_settings.enabled"
            standout
            dense
            stack-label
            label="Assrt API Token"
          />
        </article>

        <article v-if="form.subdl_settings" class="supplier-card">
          <div class="supplier-card__top">
            <div class="supplier-card__heading">
              <div class="supplier-card__title-row">
                <h4 class="supplier-card__title">SubDL</h4>
                <span class="supplier-card__link">https://subdl.com/api-doc</span>
              </div>
              <p class="supplier-card__caption">默认关闭，启用后按现有逻辑参与字幕检索。</p>
            </div>
            <q-toggle v-model="form.subdl_settings.enabled" color="primary" />
          </div>

          <q-input
            v-model="form.subdl_settings.key"
            :disable="!form.subdl_settings.enabled"
            standout
            dense
            stack-label
            label="SubDL ApiKey"
          />
        </article>

        <article v-if="form.opensubtitles_settings" class="supplier-card">
          <div class="supplier-card__top">
            <div class="supplier-card__heading">
              <div class="supplier-card__title-row">
                <h4 class="supplier-card__title">OpenSubtitles</h4>
              </div>
              <p class="supplier-card__caption">同一块内配置启用状态、API key、用户名和密码。</p>
            </div>
            <q-toggle v-model="form.opensubtitles_settings.enabled" color="primary" />
          </div>

          <div class="supplier-card__fields supplier-card__fields--double">
            <q-input
              v-model="form.opensubtitles_settings.api_key"
              :disable="!form.opensubtitles_settings.enabled"
              standout
              dense
              stack-label
              label="API key"
            />
            <q-input
              v-model="form.opensubtitles_settings.username"
              :disable="!form.opensubtitles_settings.enabled"
              standout
              dense
              stack-label
              label="用户名"
            />
            <q-input
              v-model="form.opensubtitles_settings.password"
              :disable="!form.opensubtitles_settings.enabled"
              standout
              dense
              type="password"
              stack-label
              label="密码"
            />
          </div>
        </article>

        <article v-if="form.tvsubtitles_settings" class="supplier-card supplier-card--compact">
          <div class="supplier-card__top">
            <div class="supplier-card__heading">
              <div class="supplier-card__title-row">
                <h4 class="supplier-card__title">TVsubtitles</h4>
              </div>
            </div>
            <q-toggle v-model="form.tvsubtitles_settings.enabled" color="primary" />
          </div>
        </article>

        <article v-if="form.moviesubtitles_settings" class="supplier-card supplier-card--compact">
          <div class="supplier-card__top">
            <div class="supplier-card__heading">
              <div class="supplier-card__title-row">
                <h4 class="supplier-card__title">Moviesubtitles</h4>
              </div>
              <p class="supplier-card__caption">当前仍建议按需启用。</p>
            </div>
            <q-toggle v-model="form.moviesubtitles_settings.enabled" color="primary" />
          </div>
        </article>

        <article v-if="form.subhd_settings" class="supplier-card supplier-card--compact">
          <div class="supplier-card__top">
            <div class="supplier-card__heading">
              <div class="supplier-card__title-row">
                <h4 class="supplier-card__title">SubHD</h4>
              </div>
            </div>
            <q-toggle v-model="form.subhd_settings.enabled" color="primary" />
          </div>
        </article>

        <article v-if="form.subtitlecat_settings" class="supplier-card">
          <div class="supplier-card__top">
            <div class="supplier-card__heading">
              <div class="supplier-card__title-row">
                <h4 class="supplier-card__title">SubtitleCat</h4>
              </div>
              <p class="supplier-card__caption">英文字幕下载回退默认保留；这里仅控制中文字幕远端翻译回退。</p>
            </div>
          </div>

          <div class="supplier-card__inline-toggle">
            <div class="supplier-card__inline-copy">
              <div class="supplier-card__inline-title">中文字幕远端翻译回退</div>
              <div class="supplier-card__inline-caption">默认关闭，需要用户显式确认后再启用。</div>
            </div>
            <q-toggle v-model="form.subtitlecat_settings.enable_translated_chinese_fallback" color="primary" />
          </div>
        </article>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import { toRefs } from '@vueuse/core';
import EditSubSourceBtnDialog from 'pages/settings/BtnDialogEditSubSource';
import { formModel } from 'pages/settings/use-settings';

const { subtitle_sources: form } = toRefs(formModel);
const hiddenSupplierNames = new Set(['a4k', 'zimuku']);

const visibleSuppliers = computed(() =>
  Object.values(formModel.advanced_settings?.suppliers_settings ?? {}).filter(
    (item) => item && !hiddenSupplierNames.has(item.name)
  )
);

const handleSubSourceUpdate = (item, data) => {
  formModel.advanced_settings.suppliers_settings[item.name].root_url = data.url;
  formModel.advanced_settings.suppliers_settings[item.name].daily_download_limit = data.dailyLimit;
};
</script>

<style scoped lang="scss">
.sub-source-panel {
  display: grid;
  gap: 18px;
}

.sub-source-section {
  border: 1px solid rgba(15, 23, 42, 0.05);
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.72);
  overflow: hidden;
}

.sub-source-section__header {
  padding: 18px 20px 14px;
  border-bottom: 1px solid rgba(15, 23, 42, 0.05);
  background: rgba(248, 250, 253, 0.85);
}

.sub-source-section__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #122033;
}

.sub-source-section__subtitle {
  margin: 6px 0 0;
  font-size: 13px;
  color: #6a7b91;
}

.sub-source-provider-list,
.supplier-card-list {
  display: grid;
  gap: 14px;
  padding: 18px;
}

.sub-source-provider-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  border: 1px solid rgba(15, 23, 42, 0.06);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.9);
}

.sub-source-provider-row__meta {
  min-width: 0;
  display: grid;
  gap: 4px;
}

.sub-source-provider-row__name {
  font-size: 14px;
  font-weight: 600;
  color: #122033;
}

.sub-source-provider-row__url,
.sub-source-provider-row__limit {
  font-size: 12px;
  color: #6a7b91;
  word-break: break-all;
}

.supplier-card {
  display: grid;
  gap: 14px;
  padding: 16px;
  border: 1px solid rgba(15, 23, 42, 0.06);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.92);
}

.supplier-card--compact {
  gap: 0;
}

.supplier-card__top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.supplier-card__heading {
  min-width: 0;
  display: grid;
  gap: 6px;
}

.supplier-card__title-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.supplier-card__title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: #122033;
}

.supplier-card__link {
  font-size: 12px;
  color: #74859b;
  word-break: break-all;
}

.supplier-card__caption {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  color: #66768c;
}

.supplier-card__tips {
  margin: 0;
  padding-left: 18px;
  display: grid;
  gap: 4px;
  color: #66768c;
  font-size: 13px;
}

.supplier-card__fields {
  display: grid;
  gap: 12px;
}

.supplier-card__fields--double {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.supplier-card__fields--double :deep(.q-field):last-child {
  grid-column: 1 / -1;
}

.supplier-card__inline-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  border-radius: 14px;
  background: rgba(244, 247, 251, 0.95);
}

.supplier-card__inline-copy {
  display: grid;
  gap: 4px;
}

.supplier-card__inline-title {
  font-size: 14px;
  font-weight: 600;
  color: #122033;
}

.supplier-card__inline-caption {
  font-size: 12px;
  color: #6a7b91;
}

@media (max-width: 768px) {
  .sub-source-provider-row,
  .supplier-card__top,
  .supplier-card__inline-toggle {
    flex-direction: column;
    align-items: flex-start;
  }

  .supplier-card__fields--double {
    grid-template-columns: 1fr;
  }

  .supplier-card__fields--double :deep(.q-field):last-child {
    grid-column: auto;
  }
}
</style>
