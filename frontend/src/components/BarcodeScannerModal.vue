<script setup lang="ts">
import { BrowserMultiFormatReader, type IScannerControls } from '@zxing/browser'
import { Check, StopCircle, Video } from 'lucide-vue-next'
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { validateBarcode } from '@/lib/barcode'
import { checkCameraSupport } from '@/lib/camera'

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
  // Check before touching ZXing: without a secure context navigator.mediaDevices
  // is absent, and the library fails with an unreadable TypeError.
  const support = checkCameraSupport()
  if (!support.supported) {
    cameraError.value = support.message
    scanning.value = false
    return
  }

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
  <Dialog :open="visible" @update:open="(value: boolean) => !value && closeModal()">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>Scan Barcode</DialogTitle>
        <DialogDescription>Use the camera, a USB scanner, or type the code manually.</DialogDescription>
      </DialogHeader>

      <div class="grid gap-4">
        <div class="relative overflow-hidden rounded-xl bg-brand-navy">
          <video ref="video" class="min-h-[220px] w-full object-cover" muted playsinline />
          <div v-if="scanning" class="pointer-events-none absolute inset-0 flex items-center justify-center">
            <div class="h-[120px] w-[220px] animate-pulse rounded-lg border-2 border-dashed border-brand-amber/80" />
          </div>
        </div>

        <div class="flex items-center gap-2">
          <Select v-model="selectedCamera" :disabled="scanning">
            <SelectTrigger class="flex-1">
              <SelectValue placeholder="Select camera" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="camera in cameras" :key="camera.value" :value="camera.value">
                {{ camera.label }}
              </SelectItem>
            </SelectContent>
          </Select>
          <Button v-if="!scanning" size="sm" @click="startScanner">
            <Video class="size-4" />
            Start
          </Button>
          <Button v-else size="sm" variant="secondary" @click="stopScanner">
            <StopCircle class="size-4" />
            Stop
          </Button>
        </div>

        <p
          v-if="cameraError"
          class="rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-700"
        >
          {{ cameraError }}
        </p>

        <div class="grid gap-2">
          <Label for="modal-manual-barcode">Manual or USB Scanner Input</Label>
          <Input
            id="modal-manual-barcode"
            v-model="manualValue"
            placeholder="Enter or scan EAN-13 / UPC-A"
            @keyup.enter="confirmBarcode"
          />
        </div>

        <div v-if="candidate" class="rounded-lg border bg-muted/40 p-3">
          <small class="text-xs text-muted-foreground">Detected Barcode:</small>
          <div class="text-lg font-semibold">{{ candidate }}</div>
          <p
            class="mt-1 text-xs"
            :class="validation.valid ? 'text-emerald-600' : 'text-amber-600'"
          >
            {{ validation.message }}
          </p>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="closeModal">Cancel</Button>
        <Button :disabled="!candidate" @click="confirmBarcode">
          <Check class="size-4" />
          Use Barcode
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
