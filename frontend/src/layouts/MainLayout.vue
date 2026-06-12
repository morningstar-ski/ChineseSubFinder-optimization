<template>
  <q-layout view="lHh Lpr lFf">
    <q-header elevated>
      <q-toolbar class="text-white text-primary no-wrap">
        <q-btn flat dense round color="white" icon="menu" aria-label="Menu" @click="leftDrawerOpen = !leftDrawerOpen" />
        <div class="text-h6 q-ml-sm ellipsis toolbar-title">{{ $route.meta.title }}</div>
        <q-space />

        <div class="row items-center no-wrap q-gutter-xs">
          <version-update-item>
            <q-btn
              flat
              dense
              color="white"
              icon="system_update_alt"
              :round="isCompactHeader"
              :label="isCompactHeader ? undefined : '版本升级'"
              aria-label="版本升级"
            />
          </version-update-item>

          <q-btn
            flat
            dense
            color="white"
            icon="help_outline"
            :round="isCompactHeader"
            :label="isCompactHeader ? undefined : '帮助文档'"
            aria-label="帮助文档"
            @click="openPage(PROJECT_HELP_URL)"
          />

          <q-btn
            flat
            dense
            color="white"
            icon="bug_report"
            :round="isCompactHeader"
            :label="isCompactHeader ? undefined : '问题反馈'"
            aria-label="问题反馈"
            @click="gotoGithubIssuePage"
          />
        </div>

        <q-btn-dropdown
          flat
          dense
          color="white"
          icon="account_circle"
          :round="isCompactHeader"
          :label="isCompactHeader ? undefined : userState.username"
          :dropdown-icon="isCompactHeader ? 'arrow_drop_down' : undefined"
        >
          <q-list dense style="min-width: 100px">
            <q-item clickable v-close-popup>
              <q-item-section @click="logout">退出登录</q-item-section>
            </q-item>
          </q-list>
        </q-btn-dropdown>
      </q-toolbar>
    </q-header>

    <q-drawer
      v-model="leftDrawerOpen"
      class="q-pa-md"
      :breakpoint="720"
      :width="280"
      show-if-above
      bordered
      dark
      style="background: #111729"
      content-class="bg-white"
    >
      <div class="text-h5 q-py-sm q-px-md">
        <div class="text-center">
          <img src="icons/logo.png" alt="" style="filter: invert(100%); height: 60px" />
        </div>
        <div class="q-mt-sm text-center relative-position">
          <q-badge v-if="displayVersion" align="top">{{ displayVersion }}</q-badge>
          ChineseSubFinder
        </div>
      </div>
      <q-list>
        <menu-item v-for="route in menus" :menu-info="route" :key="`${route.name}${route.path}`" />
      </q-list>
    </q-drawer>

    <q-page-container>
      <router-view />

      <notice-dialog />
    </q-page-container>
  </q-layout>
</template>

<script setup>
import routes from 'src/router/routes';
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import { LocalStorage, useQuasar } from 'quasar';
import MenuItem from 'layouts/MenuItem';
import { systemState } from 'src/store/systemState';
import { userState } from 'src/store/userState';
import AccessApi from 'src/api/AccessApi';
import VersionUpdateItem from 'components/VersionUpdateItem';
import NoticeDialog from 'components/NoticeDialog';
import { PROJECT_HELP_URL } from 'src/constants/ProjectLinks';
import { gotoGithubIssuePage } from 'src/utils/common';
import { normalizeDisplayVersion } from 'src/utils/version';

const router = useRouter();
const $q = useQuasar();

const leftDrawerOpen = ref(false);
const menus = routes.find((e) => e.path === '/').children;
const displayVersion = computed(() => normalizeDisplayVersion(systemState.systemInfo?.version));
const isCompactHeader = computed(() => $q.screen.lt.md);

const logout = async () => {
  if (userState.accessToken) {
    try {
      await AccessApi.logout();
    } catch (error) {
      void error;
    }
  }
  userState.username = '';
  userState.accessToken = undefined;
  LocalStorage.remove('token');
  router.push('/access/login');
};

const openPage = (url) => {
  window.open(url, '_blank');
};
</script>

<style scoped>
.toolbar-title {
  min-width: 0;
  max-width: min(240px, 32vw);
}
</style>
