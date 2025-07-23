<template>
  <div class="workspace-view">
    <workspace-list />
  </div>
</template>

<script setup lang="ts">
import WorkspaceList from '@/components/Workspace/WorkspaceList.vue'
</script>

<style scoped>
.workspace-view {
  height: 100vh;
  overflow: auto;
}
</style>