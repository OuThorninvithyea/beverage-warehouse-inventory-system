<script setup lang="ts">
import { BrowserMultiFormatReader, type IScannerControls } from '@zxing/browser'
import { computed, nextTick, onBeforeUnmount, ref } from 'vue'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
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
const locked = ref(false)
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
      error instanceof Error ? error.message : 'Unable to enumerate cameras.'
  }
}

async function startScanner() {
  stopScanner()
  capturedValue.value = ''
  locked.value = false
  cameraError.value = ''
  scanning.value = true

  try {
    await loadCameras()
    await nextTick()

    if (!video.value) {
      throw new Error('Camera preview is unavailable.')
    }

    const reader = new BrowserMultiFormatReader()
    controls = await reader.decodeFromVideoDevice(
      selectedCamera.value,
      video.value,
      (result) => {
        if (!result || locked.value) {
          return
        }

        capturedValue.value = result.getText()
        locked.value = true
        stopScanner()
      },
    )
  } catch (error) {
    scanning.value = false
    cameraError.value =
      error instanceof Error
        ? error.message
        : 'Camera access failed. Check HTTPS and browser permission.'
  }
}

function stopScanner() {
  controls?.stop()
  controls = null
  scanning.value = false
}

function rescan() {
  capturedValue.value = ''
  manualValue.value = ''
  void startScanner()
}

function useManualValue() {
  capturedValue.value = manualValue.value.trim()
  locked.value = true
  stopScanner()
}

onBeforeUnmount(stopScanner)
</script>

<template>
  <section class="grid gap-6">
    <div class="grid gap-2">
      <Badge variant="outline" class="w-fit border-emerald-500/30 bg-emerald-500/10 text-emerald-700">
        Week 6 device test
      </Badge>
      <h1 class="text-3xl font-semibold tracking-tight">Barcode camera validation</h1>
      <p class="text-sm text-muted-foreground">Decode and validate a barcode without changing inventory.</p>
    </div>

    <div class="grid grid-cols-2 gap-6 max-[800px]:grid-cols-1">
      <Card>
        <CardHeader>
          <CardTitle class="text-base">Camera scanner</CardTitle>
        </CardHeader>
        <CardContent class="grid gap-4">
          <video ref="video" class="min-h-[280px] w-full rounded-xl bg-brand-navy object-cover" muted playsinline />

          <div class="grid gap-2">
            <Label for="camera">Camera</Label>
            <Select v-model="selectedCamera" :disabled="scanning">
              <SelectTrigger id="camera" class="w-full">
                <SelectValue placeholder="Choose a camera" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="camera in cameras" :key="camera.value" :value="camera.value">
                  {{ camera.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="flex flex-wrap gap-2">
            <Button v-if="!scanning" @click="startScanner">Start camera</Button>
            <Button v-else variant="secondary" @click="stopScanner">Stop camera</Button>
            <Button v-if="locked" variant="outline" @click="rescan">Rescan</Button>
          </div>

          <p
            v-if="cameraError"
            class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
          >
            {{ cameraError }}
          </p>
          <small class="text-xs text-muted-foreground">
            Camera access requires HTTPS, except on <code>localhost</code>.
          </small>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle class="text-base">Captured value</CardTitle>
        </CardHeader>
        <CardContent class="grid gap-4">
          <div class="grid gap-2">
            <Label for="manual-barcode">Manual or USB scanner input</Label>
            <Input
              id="manual-barcode"
              v-model="manualValue"
              inputmode="numeric"
              autocomplete="off"
              placeholder="Scan or enter EAN-13 / UPC-A"
              @keyup.enter="useManualValue"
            />
          </div>

          <Button
            variant="secondary"
            :disabled="manualValue.trim().length === 0"
            @click="useManualValue"
          >
            Validate value
          </Button>

          <div v-if="candidate" class="grid gap-1.5 rounded-xl border bg-muted/40 p-4">
            <small class="text-xs text-muted-foreground">Captured barcode</small>
            <strong class="text-2xl tracking-[0.08em] [overflow-wrap:anywhere]">
              {{ validation.normalized }}
            </strong>
            <p class="text-sm" :class="validation.valid ? 'text-emerald-600' : 'text-amber-600'">
              {{ validation.message }}
            </p>
          </div>

          <p class="rounded-md border border-sky-500/30 bg-sky-500/10 px-3 py-2 text-sm text-sky-700">
            This screen only produces a lookup value. It cannot receive, pick, transfer or adjust inventory.
          </p>
        </CardContent>
      </Card>
    </div>
  </section>
</template>
