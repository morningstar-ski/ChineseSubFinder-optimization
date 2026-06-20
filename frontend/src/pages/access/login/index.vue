<template>
  <q-layout view="lHh Lpr fff" class="login-layout">
    <q-page-container>
      <q-page class="login-page row justify-center items-center">
        <login-bg-area />

        <q-form @submit="submit" class="login-shell">
          <section class="login-panel">
            <div class="login-panel__intro">
              <div class="login-panel__brand">
                <div class="login-panel__brand-mark">
                  <img src="icons/logo.png" alt="" class="login-panel__logo" />
                </div>
                <div>
                  <div class="login-panel__eyebrow">ChineseSubFinder</div>
                  <h1 class="login-panel__title">系统登录</h1>
                </div>
              </div>

              <div class="login-panel__status">
                <div class="login-panel__status-dot"></div>
                <span>本地字幕服务</span>
              </div>
            </div>

            <div class="login-panel__form">
              <q-input
                v-model="form.username"
                lazy-rules
                standout
                :rules="[(val) => !!val || '用户名不能为空']"
                label="用户名"
              >
                <template #prepend>
                  <q-icon name="person" />
                </template>
              </q-input>

              <q-input
                v-model="form.password"
                type="password"
                lazy-rules
                standout
                :rules="[(val) => !!val || '密码不能为空']"
                label="密码"
              >
                <template #prepend>
                  <q-icon name="lock" />
                </template>
              </q-input>
            </div>

            <q-card-actions class="login-panel__actions">
              <q-btn
                unelevated
                size="lg"
                color="primary"
                type="submit"
                :loading="submitting"
                class="full-width text-white"
                label="登录"
              />
            </q-card-actions>
          </section>
        </q-form>
      </q-page>
    </q-page-container>
  </q-layout>
</template>

<script setup>
import { reactive, ref } from 'vue';
import { SystemMessage } from 'src/utils/message';
import { useRouter } from 'vue-router';
import AccessApi from 'src/api/AccessApi';
import { LocalStorage } from 'quasar';
import { userState } from 'src/store/userState';
import LoginBgArea from 'pages/access/login/LoginBgArea';

const router = useRouter();
const form = reactive({
  username: '',
  password: '',
});

const submitting = ref(false);

const submit = async () => {
  submitting.value = true;
  const formData = { ...form };
  delete formData.confirmPassword;
  const [res, err] = await AccessApi.login(formData);
  submitting.value = false;
  if (err !== null) {
    SystemMessage.error(err.message);
    return;
  }
  const userData = {
    accessToken: res.access_token,
    username: form.username,
  };
  Object.assign(userState, userData);
  LocalStorage.set('token', userData);
  router.push('/');
};
</script>

<style scoped lang="scss">
.login-layout,
.login-page {
  min-height: 100vh;
}

.login-page {
  padding: 24px;
}

.login-shell {
  width: min(100%, 460px);
}

.login-panel {
  padding: 30px;
  border-radius: 32px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: rgba(255, 255, 255, 0.86);
  box-shadow: 0 28px 80px rgba(15, 23, 42, 0.12);
  backdrop-filter: blur(20px);
}

.login-panel__intro {
  display: grid;
  gap: 18px;
}

.login-panel__brand {
  display: flex;
  align-items: center;
  gap: 16px;
}

.login-panel__brand-mark {
  display: grid;
  width: 64px;
  height: 64px;
  place-items: center;
  border-radius: 22px;
  background: #f4f7fb;
  box-shadow: inset 0 0 0 1px rgba(15, 23, 42, 0.05);
}

.login-panel__logo {
  width: 40px;
  height: 40px;
  object-fit: contain;
}

.login-panel__eyebrow {
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  color: #7f8ea3;
}

.login-panel__title {
  margin: 8px 0 0;
  font-size: 32px;
  font-weight: 700;
  line-height: 1.1;
  color: #142033;
}

.login-panel__status {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  width: fit-content;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  background: #f4f7fb;
  color: #5a6b83;
  font-size: 13px;
  font-weight: 600;
}

.login-panel__status-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: #2bb673;
  box-shadow: 0 0 0 6px rgba(43, 182, 115, 0.12);
}

.login-panel__form {
  display: grid;
  gap: 16px;
  margin-top: 26px;
}

.login-panel__form :deep(.q-field__control) {
  background: #ffffff !important;
  color: #142033 !important;
  box-shadow: inset 0 0 0 1px rgba(15, 23, 42, 0.08), 0 10px 26px rgba(15, 23, 42, 0.05) !important;
}

.login-panel__form :deep(.q-field__native),
.login-panel__form :deep(.q-field__input),
.login-panel__form :deep(.q-field__label),
.login-panel__form :deep(.q-icon) {
  color: #142033 !important;
}

.login-panel__form :deep(.q-field__label) {
  opacity: 0.72;
}

.login-panel__form :deep(.q-field__native::placeholder),
.login-panel__form :deep(.q-field__input::placeholder) {
  color: #7f8ea3 !important;
  opacity: 1;
}

.login-panel__form :deep(.q-field--focused .q-field__label),
.login-panel__form :deep(.q-field--float .q-field__label) {
  color: #3f5f8a !important;
}

.login-panel__actions {
  margin-top: 28px;
  padding: 0;
}

@media (max-width: 599px) {
  .login-page {
    padding: 16px;
  }

  .login-panel {
    padding: 22px;
    border-radius: 28px;
  }

  .login-panel__title {
    font-size: 26px;
  }
}
</style>
