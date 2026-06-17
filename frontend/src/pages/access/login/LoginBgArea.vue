<template>
  <div class="login-bg" aria-hidden="true">
    <div class="login-bg__frame login-bg__frame--large"></div>
    <div class="login-bg__frame login-bg__frame--small"></div>
    <div class="login-bg__timeline">
      <span></span>
      <span></span>
      <span></span>
      <span></span>
      <span></span>
    </div>
  </div>
</template>

<script setup></script>

<style scoped lang="scss">
.login-bg {
  position: fixed;
  inset: 0;
  overflow: hidden;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.98) 0%, rgba(244, 247, 251, 0.98) 100%);
  z-index: -1;
}

.login-bg::before,
.login-bg::after {
  content: '';
  position: absolute;
  inset: 0;
}

.login-bg::before {
  background: linear-gradient(180deg, rgba(22, 119, 255, 0.04) 0%, rgba(22, 119, 255, 0) 32%),
    repeating-linear-gradient(
      90deg,
      rgba(255, 255, 255, 0) 0,
      rgba(255, 255, 255, 0) 104px,
      rgba(20, 32, 51, 0.03) 104px,
      rgba(20, 32, 51, 0.03) 105px
    );
}

.login-bg::after {
  background: linear-gradient(180deg, rgba(255, 255, 255, 0) 0%, rgba(255, 255, 255, 0.56) 100%);
}

.login-bg__frame {
  position: absolute;
  border-radius: 32px;
  border: 1px solid rgba(22, 119, 255, 0.08);
  background: rgba(255, 255, 255, 0.52);
  box-shadow: 0 24px 80px rgba(15, 23, 42, 0.06);
  backdrop-filter: blur(16px);
}

.login-bg__frame--large {
  top: 9%;
  left: 8%;
  width: min(36vw, 440px);
  height: min(54vh, 520px);
  animation: drift 16s ease-in-out infinite;
}

.login-bg__frame--small {
  right: 10%;
  bottom: 10%;
  width: min(24vw, 280px);
  height: min(28vh, 260px);
  animation: drift 18s ease-in-out infinite reverse;
}

.login-bg__timeline {
  position: absolute;
  left: 12%;
  right: 12%;
  bottom: 12%;
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 14px;
}

.login-bg__timeline span {
  display: block;
  height: 10px;
  border-radius: 999px;
  background: linear-gradient(90deg, rgba(22, 119, 255, 0.1), rgba(95, 184, 255, 0.3));
}

@keyframes drift {
  0%,
  100% {
    transform: translate3d(0, 0, 0);
  }

  50% {
    transform: translate3d(0, -10px, 0);
  }
}

@media (max-width: 780px) {
  .login-bg__frame--large {
    left: 6%;
    width: 56vw;
  }

  .login-bg__frame--small {
    right: 6%;
    width: 34vw;
  }

  .login-bg__timeline {
    left: 8%;
    right: 8%;
  }
}
</style>
