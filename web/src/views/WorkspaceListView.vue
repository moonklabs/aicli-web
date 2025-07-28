<template>
  <div class="workspace-list-view">
    <WorkspaceList />
  </div>
</template>

<script setup lang="ts">
import WorkspaceList from '@/components/Workspace/WorkspaceList.vue'
</script>

<style scoped>
.workspace-list-view {
  height: 100vh;
  overflow: auto;
}
</style>