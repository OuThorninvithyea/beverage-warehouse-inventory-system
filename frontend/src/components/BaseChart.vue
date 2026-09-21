<script setup lang="ts">
import {
  Chart as ChartJS,
  type ChartData,
  type ChartOptions,
  type ChartType,
  registerables,
} from 'chart.js'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

ChartJS.register(...registerables)

const props = defineProps<{
  type: ChartType
  data: ChartData
  options?: ChartOptions
}>()

const canvas = ref<HTMLCanvasElement | null>(null)
let chart: ChartJS | null = null

function render() {
  if (!canvas.value) return
  chart?.destroy()
  chart = new ChartJS(canvas.value, {
    type: props.type,
    data: props.data as ChartData,
    options: props.options,
  })
}

onMounted(render)
watch(() => props.data, render, { deep: true })

onBeforeUnmount(() => {
  chart?.destroy()
  chart = null
})
</script>

<template>
  <canvas ref="canvas" class="size-full" />
</template>
