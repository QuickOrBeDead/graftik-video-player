<script setup lang="ts">
defineProps({
  title: String
})

function closeWindow() {
  window.api.closeWindow()
}
</script>
<template>
  <div class="title-bar drag">
    <div class="title text-white">
      <slot name="icon">
        <i class="bi bi-gear"></i>
      </slot>
      <span class="ms-1">{{ title }}</span>
    </div>
    <div class="no-drag">
      <button class="close-btn" @click="closeWindow()">
        <i class="bi bi-x-lg"></i>
      </button>
    </div>
  </div>
</template>
<style scoped>
.drag,
[draggable] {
  -webkit-app-region: drag;
}
.no-drag {
  -webkit-app-region: no-drag;
}
.title-bar {
  background-color: #333333;
  padding: 5px 10px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid #444444;
}
.title-bar .title {
  display: flex;
  align-items: center;
}
.title-bar .title i {
  font-size: 1rem;
}
.title-bar span {
  font-size: 0.85rem;
  line-height: 1rem;
}
.title-bar .close-btn {
  background: none;
  border: none;
  color: #ffffff;
  cursor: pointer;
  font-size: 0.65rem;
  line-height: 0.75rem;
}
.title-bar .close-btn:hover {
  color: #777777;
}
</style>
