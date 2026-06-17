<template>
  <div v-if="menuInfo.meta && menuInfo.meta.title" class="menu-block">
    <q-expansion-item
      v-if="menuInfo.children && menuInfo.children.length"
      :label="menuInfo.meta.title"
      :icon="menuInfo.meta.icon"
      :default-opened="defaultOpened"
      class="menu-expansion"
      expand-icon-class="menu-expansion__arrow"
      header-class="menu-expansion__header"
    >
      <div class="menu-expansion__children">
        <menu-item v-for="subMenu in menuInfo.children" :menu-info="subMenu" :key="subMenu.name" />
      </div>
    </q-expansion-item>

    <q-item
      v-else
      :to="{ name: menuInfo.name }"
      :active="$route.name === menuInfo.name"
      clickable
      v-ripple
      class="menu-link"
    >
      <q-item-section v-if="menuInfo.meta.icon" avatar class="menu-link__avatar">
        <q-icon :name="menuInfo.meta.icon" />
      </q-item-section>
      <q-item-section class="menu-link__label">{{ menuInfo.meta.title }}</q-item-section>
    </q-item>
  </div>
</template>

<script>
import { defineComponent } from 'vue';

export default defineComponent({
  name: 'MenuItem',
  props: {
    menuInfo: {
      type: Object,
      required: true,
    },
  },
  computed: {
    defaultOpened() {
      return this.$route.matched.some((e) => e.name === this.menuInfo.name);
    },
  },
});
</script>

<style scoped lang="scss">
.menu-block + .menu-block {
  margin-top: 6px;
}

.menu-link,
:deep(.menu-expansion__header) {
  min-height: 48px;
  border-radius: 16px;
  color: #566781;
  transition: background-color 0.2s ease, color 0.2s ease, transform 0.2s ease;
}

.menu-link:hover,
:deep(.menu-expansion__header:hover) {
  background: #f3f6fb;
  color: #142033;
}

.menu-link.q-router-link--active,
:deep(.q-expansion-item--expanded > .menu-expansion__header) {
  background: rgba(22, 119, 255, 0.1);
  color: #1677ff;
}

.menu-link__avatar,
:deep(.menu-expansion__header .q-item__section--avatar) {
  min-width: 36px;
  color: inherit;
}

.menu-link__label,
:deep(.menu-expansion__header .q-item__label) {
  font-weight: 600;
}

.menu-expansion {
  border-radius: 18px;
}

.menu-expansion__children {
  display: grid;
  gap: 4px;
  padding: 6px 0 0 8px;
}

:deep(.menu-expansion__arrow) {
  color: inherit;
}
</style>
