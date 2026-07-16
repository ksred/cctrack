<template>
  <template v-for="family in families" :key="family.id">
    <ProjectFamilyRow
      :family-id="family.id"
      :display-name="family.displayName"
      :rollup="family.rollup"
      :expanded="headerExpanded(family)"
      @toggle="onHeaderToggle(family)"
    />

    <!-- Singleton family: header expands straight into that project's sessions. -->
    <template v-if="family.members.length === 1">
      <template v-if="expanded.has(family.members[0].group.project)">
        <tr v-if="childLoading.has(family.members[0].group.project)" class="loading-row">
          <td></td>
          <td colspan="5">Loading sessions…</td>
        </tr>
        <SessionRow
          v-for="(session, i) in (childSessions.get(family.members[0].group.project) || [])"
          :key="session.id"
          :session="session"
          :rank="i + 1"
          :depth="0"
          show-started
          subordinate
          @select="emit('select', $event)"
        />
      </template>
    </template>

    <!-- Multi-member family: header collapses the tree; each member expands sessions. -->
    <template v-else-if="isFamilyOpen(family.id)">
      <template v-for="node in family.members" :key="node.group.project">
        <ProjectGroupRow
          :group="node.group"
          :display-name="node.displayName"
          :depth="node.depth"
          :expanded="expanded.has(node.group.project)"
          @toggle="emit('toggle', $event)"
        />
        <template v-if="expanded.has(node.group.project)">
          <tr v-if="childLoading.has(node.group.project)" class="loading-row">
            <td></td>
            <td colspan="5">Loading sessions…</td>
          </tr>
          <SessionRow
            v-for="(session, i) in (childSessions.get(node.group.project) || [])"
            :key="session.id"
            :session="session"
            :rank="i + 1"
            :depth="node.depth"
            show-started
            subordinate
            @select="emit('select', $event)"
          />
        </template>
      </template>
    </template>
  </template>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { Session } from '../../types'
import type { ProjectFamily } from '../../composables/useWorktreeNesting'
import ProjectFamilyRow from './ProjectFamilyRow.vue'
import ProjectGroupRow from './ProjectGroupRow.vue'
import SessionRow from './SessionRow.vue'

const props = defineProps<{
  families: ProjectFamily[]
  expanded: Set<string>
  childSessions: Map<string, Session[]>
  childLoading: Set<string>
}>()

const emit = defineEmits<{
  toggle: [project: string]
  select: [id: string]
}>()

// Multi-member families start expanded. Collapsing is local UI state only.
const collapsed = ref<Set<string>>(new Set())

function isFamilyOpen(id: string) {
  return !collapsed.value.has(id)
}

function toggleFamily(id: string) {
  const next = new Set(collapsed.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  collapsed.value = next
}

function headerExpanded(family: ProjectFamily) {
  if (family.members.length === 1) {
    return props.expanded.has(family.members[0].group.project)
  }
  return isFamilyOpen(family.id)
}

function onHeaderToggle(family: ProjectFamily) {
  if (family.members.length === 1) {
    emit('toggle', family.members[0].group.project)
    return
  }
  toggleFamily(family.id)
}
</script>
