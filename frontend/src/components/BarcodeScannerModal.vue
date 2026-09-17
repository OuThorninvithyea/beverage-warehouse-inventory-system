<script setup lang="ts">
import { BrowserMultiFormatReader, type IScannerControls } from '@zxing/browser'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import Select from 'primevue/select'
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'

import { validateBarcode } from '@/lib/barcode'

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'select', barcode: string): void
}>()

interface CameraOption {
  label: string
  value: string
}

const video = ref<HTMLVideoElement | null>(null)
const manualValue = ref('')
const capturedValue = ref('')
const cameras = ref<CameraOption[]>([])
const selectedCamera = ref<string>()
const cameraError = ref('')
const scanning = ref(false)
let controls: IScannerControls | null = null

const candidate = computed(() => capturedValue.value || manualValue.value)
const validation = computed(() => validateBarcode(candidate.value))

async function loadCameras() {
  try {
    const devices = await BrowserMultiFormatReader.listVideoInputDevices()
    cameras.value = devices.map((device, index) => ({
      label: device.label || `Camera ${index + 1}`,
      value: device.deviceId,
    }))
    selectedCamera.value ??= cameras.value.at(-1)?.value
  } catch (error) {
    cameraError.value =
      error instanceof Error ? error.message : 'Unable to access cameras.'
  }
}

async function startScanner() {
  stopScanner()
  capturedValue.value = ''
  cameraError.value = ''
  scanning.value = true

  try {
    await loadCameras()
    await nextTick()

    if (!video.value) {
      throw new Error('Camera preview element missing.')
    }

    const reader = new BrowserMultiFormatReader()
    controls = await reader.decodeFromVideoDevice(
      selectedCamera.value,
      video.value,
      (result) => {
        if (!result) return
        capturedValue.value = result.getText()
        stopScanner()
      },
    )
  } catch (error) {
    scanning.value = false
    cameraError.value =
      error instanceof Error
        ? error.message
        : 'Camera access failed.'
  }
}

function stopScanner() {
  controls?.stop()
  controls = null
  scanning.value = false
}

function confirmBarcode() {
  const barcodeToUse = validation.value.normalized || candidate.value.trim()
  if (barcodeToUse) {
    emit('select', barcodeToUse)
    closeModal()
  }
}

function closeModal() {
  stopScanner()
  manualValue.value = ''
  capturedValue.value = ''
  emit('update:visible', false)
}

watch(
  () => props.visible,
  (newVal) => {
    if (newVal) {
      void startScanner()
    } else {
      stopScanner()
    }
  },
)

onBeforeUnmount(stopScanner)
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    header="Scan Barcode"
    :style="{ width: '90vw', maxWidth: '520px' }"
    @update:visible="closeModal"
  >
    <div class="grid gap-4">
      <div class="relative overflow-hidden rounded-[12px] bg-[#091a2d]">
        <video
          ref="video"
          class="min-h-[220px] w-full object-cover"
          muted
          playsinline
        />
        <div
          v-if="scanning"
          class="pointer-events-none absolute inset-0 flex items-center justify-center"
        >
          <div class="h-[120px] w-[220px] rounded-[8px] border-2 border-dashed border-brand-amber/80 shadow-[0_0_20px_rgba(230,165,0,0.3)] animate-pulse" />
        </div>
      </div>

      <div class="flex items-center gap-2">
        <Select
          v-model="selectedCamera"
          :options="cameras"
          option-label="label"
          option-value="value"
          placeholder="Select camera"
          class="w-full"
          :disabled="scanning"
        />
        <Button
          v-if="!scanning"
          label="Start"
          icon="pi pi-video"
          size="small"
          @click="startScanner"
        />
        <Button
          v-else
          label="Stop"
          severity="secondary"
          icon="pi pi-[#091a2d]"
          size="small"
          @click="stopScanner"
        />
      </div>

      <Message v-if="cameraError" severity="warn" class="text-xs">
        {{ cameraError }}
      </Message>

      <div class="grid gap-1">
        <label for="modal-manual-barcode" class="text-xs font-semibold text-brand-muted">Manual or USB Scanner Input</label>
        <div class="flex gap-2">
          <InputText
            id="modal-manual-barcode"
            v-model="manualValue"
            placeholder="Enter or scan EAN-13 / UPC-A"
            class="w-full"
            @keyup.enter="confirmBarcode"
          />
        </div>
      </div>

      <div v-if="candidate" class="rounded-[8px] bg-brand-surface p-3 text-xs">
        <small class="text-brand-muted">Detected Barcode:</small>
        <div class="text-lg font-bold text-brand-navy">{{ candidate }}</div>
        <Message :severity="validation.valid ? 'success' : 'warn'" class="mt-1">
          {{ validation.message }}
        </Message>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" outlined @click="closeModal" />
        <Button
          label="Use Barcode"
          icon="pi pi-check"
          :disabled="!candidate"
          @click="confirmBarcode"
        />
      </div>
    </template>
  </Dialog>
</template>
