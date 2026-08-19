<script setup lang="ts">
import { BrowserMultiFormatReader, type IScannerControls } from '@zxing/browser'
import Button from 'primevue/button'
import Card from 'primevue/card'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import Select from 'primevue/select'
import { computed, nextTick, onBeforeUnmount, ref } from 'vue'

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
  <section>
    <div class="mb-6 flex items-start justify-between">
      <div>
        <span class="inline-flex rounded-full bg-[#dff8eb] px-[0.65rem] py-[0.35rem] text-[0.8rem] font-bold text-[#176c43]">Week 6 device test</span>
        <h1 class="mb-[0.35rem] mt-3 text-[clamp(1.8rem,4vw,2.6rem)]">Barcode camera validation</h1>
        <p class="m-0 text-brand-muted">Decode and validate a barcode without changing inventory.</p>
      </div>
    </div>

    <div class="grid grid-cols-2 gap-4 max-[800px]:grid-cols-1">
      <Card>
        <template #title>Camera scanner</template>
        <template #content>
          <div class="grid gap-[0.85rem]">
            <video ref="video" class="min-h-[280px] w-full rounded-[12px] bg-[#091a2d] object-cover" muted playsinline />

            <label for="camera">Camera</label>
            <Select
              id="camera"
              v-model="selectedCamera"
              :options="cameras"
              option-label="label"
              option-value="value"
              placeholder="Choose a camera"
              :disabled="scanning"
            />

            <div class="flex flex-wrap gap-[0.65rem]">
              <Button v-if="!scanning" label="Start camera" @click="startScanner" />
              <Button
                v-else
                label="Stop camera"
                severity="secondary"
                @click="stopScanner"
              />
              <Button
                v-if="locked"
                label="Rescan"
                severity="secondary"
                outlined
                @click="rescan"
              />
            </div>

            <Message v-if="cameraError" severity="error">{{ cameraError }}</Message>
            <small>
              Camera access requires HTTPS, except on <code>localhost</code>.
            </small>
          </div>
        </template>
      </Card>

      <Card>
        <template #title>Captured value</template>
        <template #content>
          <div class="grid gap-[0.85rem]">
            <label for="manual-barcode">Manual or USB scanner input</label>
            <InputText
              id="manual-barcode"
              v-model="manualValue"
              inputmode="numeric"
              autocomplete="off"
              placeholder="Scan or enter EAN-13 / UPC-A"
              @keyup.enter="useManualValue"
            />
            <Button
              label="Validate value"
              severity="secondary"
              :disabled="manualValue.trim().length === 0"
              @click="useManualValue"
            />

            <div v-if="candidate" class="grid gap-[0.4rem] rounded-[12px] bg-brand-surface p-4">
              <small>Captured barcode</small>
              <strong class="[overflow-wrap:anywhere] text-[1.4rem] tracking-[0.08em] text-brand-navy">{{ validation.normalized }}</strong>
              <Message :severity="validation.valid ? 'success' : 'warn'">
                {{ validation.message }}
              </Message>
            </div>

            <Message severity="info">
              This screen only produces a lookup value. It cannot receive, pick,
              transfer or adjust inventory.
            </Message>
          </div>
        </template>
      </Card>
    </div>
  </section>
</template>
