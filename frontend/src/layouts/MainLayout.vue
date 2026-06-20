<template>
  <q-layout view="lHh Lpr lFf" class="app-layout">
    <q-header class="app-header">
      <q-toolbar class="app-toolbar">
        <div class="app-toolbar__left">
          <q-btn
            flat
            dense
            round
            icon="menu"
            aria-label="Menu"
            class="app-toolbar__menu"
            @click="leftDrawerOpen = !leftDrawerOpen"
          />
          <div class="app-toolbar__title-group">
            <div class="app-toolbar__eyebrow">ChineseSubFinder</div>
            <div class="app-toolbar__title">{{ $route.meta.title }}</div>
          </div>
        </div>

        <div class="app-toolbar__right">
          <version-update-item>
            <q-item clickable class="app-toolbar__link">
              <q-item-section class="q-px-sm"> 版本更新 </q-item-section>
            </q-item>
          </version-update-item>

          <q-item clickable class="app-toolbar__link" @click="openPage(PROJECT_HELP_URL)">
            <q-item-section> 帮助文档 </q-item-section>
          </q-item>

          <BugReportItem />

          <q-btn-dropdown
            flat
            no-caps
            icon="account_circle"
            class="app-toolbar__account"
            content-class="app-toolbar__account-menu"
          >
            <template #label>
              <span class="app-toolbar__username">{{ userState.username }}</span>
            </template>

            <q-list dense style="min-width: 120px">
              <q-item clickable v-close-popup @click="logout">
                <q-item-section>退出登录</q-item-section>
              </q-item>
            </q-list>
          </q-btn-dropdown>
        </div>
      </q-toolbar>
    </q-header>

    <q-drawer v-model="leftDrawerOpen" class="app-drawer" :breakpoint="880" :width="304" show-if-above bordered>
      <div class="app-drawer__inner">
        <div class="app-brand">
          <div class="app-brand__logo-wrap">
            <img src="icons/logo.png" alt="" class="app-brand__logo" />
          </div>
          <div class="app-brand__copy">
            <div class="app-brand__name">ChineseSubFinder</div>
            <div class="app-brand__meta">
              <q-badge v-if="displayVersion" class="app-brand__badge">{{ displayVersion }}</q-badge>
              <span>本地字幕服务</span>
            </div>
          </div>
        </div>

        <q-list class="app-drawer__menu">
          <menu-item v-for="route in menus" :menu-info="route" :key="`${route.name}${route.path}`" />
        </q-list>
      </div>
    </q-drawer>

    <q-page-container class="app-page-container">
      <router-view />
    </q-page-container>
  </q-layout>
</template>

<script setup>
import routes from 'src/router/routes';
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import MenuItem from 'layouts/MenuItem';
import { systemState } from 'src/store/systemState';
import { userState } from 'src/store/userState';
import { LocalStorage } from 'quasar';
import AccessApi from 'src/api/AccessApi';
import BugReportItem from 'layouts/BugReportItem';
import VersionUpdateItem from 'components/VersionUpdateItem';
import { PROJECT_HELP_URL } from 'src/constants/ProjectLinks';
import { normalizeDisplayVersion } from 'src/utils/version';

const router = useRouter();

const leftDrawerOpen = ref(false);
const menus = routes.find((e) => e.path === '/').children;
const displayVersion = computed(() => normalizeDisplayVersion(systemState.systemInfo?.version));

const logout = () => {
  userState.username = '';
  userState.accessToken = undefined;
  LocalStorage.remove('token');
  AccessApi.logout();
  router.push('/access/login');
};

const openPage = (url) => {
  window.open(url, '_blank');
};
</script>

<style scoped lang="scss">
.app-layout {
  min-height: 100vh;
}

.app-header {
  background: transparent;
}

.app-toolbar {
  margin: 16px 16px 0;
  min-height: 72px;
  padding: 10px 14px 10px 12px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.86);
  box-shadow: 0 18px 48px rgba(15, 23, 42, 0.08);
  backdrop-filter: blur(18px);
}

.app-toolbar__left,
.app-toolbar__right {
  display: flex;
  align-items: center;
}

.app-toolbar__left {
  gap: 10px;
  min-width: 0;
}

.app-toolbar__right {
  margin-left: auto;
  gap: 8px;
}

.app-toolbar__menu {
  background: #f3f6fb;
  color: #142033;
}

.app-toolbar__title-group {
  min-width: 0;
}

.app-toolbar__eyebrow {
  font-size: 11px;
  font-weight: 700;
  line-height: 1;
  text-transform: uppercase;
  color: #7f8ea3;
}

.app-toolbar__title {
  margin-top: 6px;
  font-size: 21px;
  font-weight: 700;
  line-height: 1.1;
  color: #142033;
}

.app-toolbar__link {
  min-height: 40px;
  border-radius: 14px;
  color: #5b6c83;
}

.app-toolbar__link:hover {
  background: #f3f6fb;
  color: #142033;
}

.app-toolbar__account {
  min-height: 42px;
  padding: 0 4px 0 8px;
  border-radius: 16px;
  background: #f3f6fb;
  color: #142033;
}

.app-toolbar__username {
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
}

.app-drawer {
  background: transparent;
}

.app-drawer__inner {
  height: calc(100% - 32px);
  margin: 16px;
  padding: 16px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 28px;
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 24px 60px rgba(15, 23, 42, 0.08);
  backdrop-filter: blur(18px);
}

.app-brand {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px;
  border-radius: 22px;
  background: linear-gradient(180deg, rgba(244, 247, 251, 0.96) 0%, rgba(255, 255, 255, 0.96) 100%);
  border: 1px solid rgba(22, 119, 255, 0.1);
}

.app-brand__logo-wrap {
  display: grid;
  width: 52px;
  height: 52px;
  place-items: center;
  border-radius: 18px;
  background: #ffffff;
  box-shadow: inset 0 0 0 1px rgba(15, 23, 42, 0.05);
}

.app-brand__logo {
  width: 34px;
  height: 34px;
  object-fit: contain;
}

.app-brand__copy {
  min-width: 0;
}

.app-brand__name {
  font-size: 18px;
  font-weight: 700;
  color: #142033;
}

.app-brand__meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
  font-size: 12px;
  color: #7f8ea3;
}

.app-brand__badge {
  background: rgba(22, 119, 255, 0.12);
  color: #1677ff;
}

.app-drawer__menu {
  margin-top: 18px;
}

.app-page-container {
  padding-top: 6px;
}

@media (max-width: 1023px) {
  .app-toolbar {
    margin: 12px 12px 0;
    min-height: 64px;
  }

  .app-toolbar__title {
    font-size: 18px;
  }

  .app-toolbar__right {
    gap: 4px;
  }
}

@media (max-width: 599px) {
  .app-toolbar {
    padding: 10px;
  }

  .app-toolbar__right .app-toolbar__link {
    display: none;
  }

  .app-toolbar__username {
    max-width: 88px;
  }
}
</style>
