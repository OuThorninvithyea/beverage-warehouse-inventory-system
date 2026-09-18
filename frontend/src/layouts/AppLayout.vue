<script setup lang="ts">
import Button from 'primevue/button'
import ConfirmDialog from 'primevue/confirmdialog'
import Popover from 'primevue/popover'
import SpeedDial from 'primevue/speeddial'
import Toast from 'primevue/toast'
import { useToast } from 'primevue/usetoast'
import { computed, onMounted, ref } from 'vue'
import { RouterLink, RouterView, useRouter } from 'vue-router'

import AdjustFormDialog from '@/components/AdjustFormDialog.vue'
import BarcodeScannerModal from '@/components/BarcodeScannerModal.vue'
import PickFormDialog from '@/components/PickFormDialog.vue'
import ReceiveFormDialog from '@/components/ReceiveFormDialog.vue'
import TransferFormDialog from '@/components/TransferFormDialog.vue'
import type { Lot } from '@/api/inventory'
import { useAuthStore } from '@/stores/auth'
import { useCatalogStore } from '@/stores/catalog'
import { useInventoryStore } from '@/stores/inventory'
import { useWarehousesStore } from '@/stores/warehouses'

const auth = useAuthStore()
const router = useRouter()
const toast = useToast()
const catalogStore = useCatalogStore()
const warehouseStore = useWarehousesStore()
const inventoryStore = useInventoryStore()

const mobileMenuOpen = ref(false)
const isDarkMode = ref(false)
const notificationPanel = ref()
const lotsByProduct = ref<Map<string, Lot[]>>(new Map())

const receiveVisible = ref(false)
const pickVisible = ref(false)
const transferVisible = ref(false)
const adjustVisible = ref(false)
const scannerVisible = ref(false)

const isAdmin = computed(() => auth.user?.role === 'admin')
const canReceiveOrPick = computed(() => {
  return auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager' || auth.user?.role === 'picker'
})

const activeAlerts = computed(() => {
  return inventoryStore.balances
    .map((b) => {
      const product = catalogStore.products.find((p) => p.id === b.product_id)
      const lot = b.lot_id ? (lotsByProduct.value.get(b.product_id) ?? []).find((l) => l.id === b.lot_id) : undefined
      return {
        id: b.id,
        product_name: product?.name ?? 'Item',
        quantity: b.quantity,
        expiration_date: lot?.expiration_date ?? null,
      }
    })
    .filter((b) => {
      const isLow = parseFloat(b.quantity) < 10
      let isExp = false
      if (b.expiration_date) {
        const diffDays = Math.ceil((new Date(b.expiration_date).getTime() - new Date().getTime()) / (1000 * 3600 * 24))
        isExp = diffDays <= 30
      }
      return isLow || isExp
    })
})

onMounted(() => {
  const storedTheme = localStorage.getItem('bwims_theme')
  if (storedTheme === 'dark') {
    isDarkMode.value = true
    document.documentElement.classList.add('bwims-dark')
  }

  // Preload data the quick-action dialogs (Receive/Pick/Transfer/Adjust) need,
  // so they aren't empty if the user hasn't visited Inventory/Movements yet.
  void catalogStore.fetchProducts()
  void warehouseStore.fetchAllLocations()
  inventoryStore.fetchBalances().then(async () => {
    const productIds = new Set(inventoryStore.balances.filter((b) => b.lot_id).map((b) => b.product_id))
    for (const productId of productIds) {
      const lots = await inventoryStore.fetchLots(productId)
      lotsByProduct.value.set(productId, lots)
    }
  })
})

function toggleTheme() {
  isDarkMode.value = !isDarkMode.value
  if (isDarkMode.value) {
    document.documentElement.classList.add('bwims-dark')
    localStorage.setItem('bwims_theme', 'dark')
  } else {
    document.documentElement.classList.remove('bwims-dark')
    localStorage.setItem('bwims_theme', 'light')
  }
}

const speedDialItems = computed(() => {
  const items = [
    {
      label: 'Scan Barcode',
      icon: 'pi pi-camera',
      command: () => {
        scannerVisible.value = true
      },
    },
  ]

  if (canReceiveOrPick.value) {
    items.unshift(
      {
        label: 'Receive Stock',
        icon: 'pi pi-download',
        command: () => {
          receiveVisible.value = true
        },
      },
      {
        label: 'FEFO Pick',
        icon: 'pi pi-upload',
        command: () => {
          pickVisible.value = true
        },
      },
    )
  }

  if (isAdmin.value || auth.user?.role === 'warehouse_manager') {
    items.push(
      {
        label: 'Transfer',
        icon: 'pi pi-arrows-h',
        command: () => {
          transferVisible.value = true
        },
      },
      {
        label: 'Adjust',
        icon: 'pi pi-sliders-h',
        command: () => {
          adjustVisible.value = true
        },
      },
    )
  }

  return items
})

async function signOut() {
  await auth.signOut()
  await router.push({ name: 'login' })
}

function closeMobileMenu() {
  mobileMenuOpen.value = false
}

function onBarcodeScanned(code: string) {
  toast.add({
    severity: 'info',
    summary: 'Barcode Scanned',
    detail: `Code: ${code}`,
    life: 3000,
  })
  void router.push({ name: 'products', query: { search: code } })
}

function onMovementSuccess(msg: string) {
  toast.add({
    severity: 'success',
    summary: 'Operation Complete',
    detail: msg,
    life: 4000,
  })
  void inventoryStore.fetchBalances()
  void inventoryStore.fetchMovements()
}
</script>

<template>
  <div class="grid min-h-screen grid-cols-[260px_minmax(0,1fr)] max-[900px]:grid-cols-1">
    <Toast />
    <ConfirmDialog />

    <!-- Mobile top bar header -->
    <div class="hidden min-[901px]:hidden flex items-center justify-between bg-brand-navy p-4 text-white max-[900px]:flex">
      <div class="flex items-center gap-3">
        <span class="grid h-8 w-8 place-items-center rounded-lg bg-brand-amber font-extrabold text-brand-navy text-xs">BW</span>
        <strong class="text-sm">BWIMS</strong>
      </div>
      <Button
        icon="pi pi-bars"
        severity="secondary"
        text
        class="!text-white"
        @click="mobileMenuOpen = !mobileMenuOpen"
      />
    </div>

    <!-- Sidebar navigation -->
    <aside
      class="flex flex-col gap-6 bg-brand-navy p-5 text-[#f7f9fc] max-[900px]:p-4"
      :class="{ 'max-[900px]:hidden': !mobileMenuOpen }"
    >
      <div class="flex items-center gap-3 px-2">
        <span class="grid h-[42px] w-[42px] place-items-center rounded-[12px] bg-brand-amber font-extrabold text-brand-navy text-lg shadow-sm">
          BW
        </span>
        <div class="grid">
          <strong class="text-base tracking-tight text-white">BWIMS</strong>
          <small class="text-xs text-[#a8bdd5]">Beverage Warehouse Control</small>
        </div>
      </div>

      <nav aria-label="Primary navigation" class="grid gap-1">
        <RouterLink
          to="/"
          class="flex items-center gap-3 rounded-[10px] px-[0.9rem] py-[0.7rem] text-sm font-medium text-[#d0e0f2] transition-colors hover:bg-white/10 [&.router-link-exact-active]:bg-brand-amber [&.router-link-exact-active]:font-bold [&.router-link-exact-active]:text-brand-navy"
          @click="closeMobileMenu"
        >
          <i class="pi pi-th-large text-base" />
          <span>Dashboard</span>
        </RouterLink>

        <RouterLink
          to="/inventory"
          class="flex items-center gap-3 rounded-[10px] px-[0.9rem] py-[0.7rem] text-sm font-medium text-[#d0e0f2] transition-colors hover:bg-white/10 [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-navy"
          @click="closeMobileMenu"
        >
          <i class="pi pi-database text-base" />
          <span>Inventory Stock</span>
        </RouterLink>

        <RouterLink
          to="/movements"
          class="flex items-center gap-3 rounded-[10px] px-[0.9rem] py-[0.7rem] text-sm font-medium text-[#d0e0f2] transition-colors hover:bg-white/10 [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-navy"
          @click="closeMobileMenu"
        >
          <i class="pi pi-truck text-base" />
          <span>Stock Movements</span>
        </RouterLink>

        <RouterLink
          to="/products"
          class="flex items-center gap-3 rounded-[10px] px-[0.9rem] py-[0.7rem] text-sm font-medium text-[#d0e0f2] transition-colors hover:bg-white/10 [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-navy"
          @click="closeMobileMenu"
        >
          <i class="pi pi-box text-base" />
          <span>Products Catalog</span>
        </RouterLink>

        <RouterLink
          to="/categories"
          class="flex items-center gap-3 rounded-[10px] px-[0.9rem] py-[0.7rem] text-sm font-medium text-[#d0e0f2] transition-colors hover:bg-white/10 [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-navy"
          @click="closeMobileMenu"
        >
          <i class="pi pi-tags text-base" />
          <span>Categories</span>
        </RouterLink>

        <RouterLink
          to="/warehouses"
          class="flex items-center gap-3 rounded-[10px] px-[0.9rem] py-[0.7rem] text-sm font-medium text-[#d0e0f2] transition-colors hover:bg-white/10 [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-navy"
          @click="closeMobileMenu"
        >
          <i class="pi pi-building text-base" />
          <span>Warehouses</span>
        </RouterLink>

        <RouterLink
          v-if="isAdmin"
          to="/users"
          class="flex items-center gap-3 rounded-[10px] px-[0.9rem] py-[0.7rem] text-sm font-medium text-[#d0e0f2] transition-colors hover:bg-white/10 [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-navy"
          @click="closeMobileMenu"
        >
          <i class="pi pi-users text-base" />
          <span>User Management</span>
        </RouterLink>

        <RouterLink
          to="/barcode-test"
          class="flex items-center gap-3 rounded-[10px] px-[0.9rem] py-[0.7rem] text-sm font-medium text-[#d0e0f2] transition-colors hover:bg-white/10 [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-navy"
          @click="closeMobileMenu"
        >
          <i class="pi pi-camera text-base" />
          <span>Barcode Test</span>
        </RouterLink>
      </nav>

      <div class="mt-auto grid gap-1 rounded-[12px] border border-white/10 bg-white/5 p-3 text-xs text-[#dce7f2] max-[900px]:hidden">
        <div class="flex items-center justify-between text-[#a8bdd5] font-semibold">
          <span>System Status</span>
          <span class="inline-flex items-center gap-1 text-[10px] text-emerald-400">
            <span class="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse" /> Live
          </span>
        </div>
        <p class="m-0 text-[11px] text-[#a8bdd5]/90 mt-1">Multi-warehouse stock control, FEFO picking, and audit trail active.</p>
      </div>
    </aside>

    <!-- Main content container -->
    <div class="min-w-0 flex flex-col bg-brand-surface relative">
      <header class="flex min-h-[70px] items-center justify-between border-b border-[#dde4ee] bg-white px-8 py-3 max-[520px]:px-4">
        <div class="grid gap-0.5">
          <small class="uppercase tracking-wider text-[10px] font-bold text-brand-muted">Beverage Distributor</small>
          <strong class="text-sm font-extrabold text-brand-navy max-[520px]:text-xs">Inventory Management System</strong>
        </div>

        <div class="flex items-center gap-3">
          <Button
            :icon="isDarkMode ? 'pi pi-sun' : 'pi pi-moon'"
            severity="secondary"
            text
            rounded
            size="small"
            :title="isDarkMode ? 'Switch to Light Mode' : 'Switch to Dark Mode'"
            @click="toggleTheme"
          />

          <div class="relative">
            <Button
              icon="pi pi-bell"
              severity="secondary"
              text
              rounded
              size="small"
              title="Notifications & Alerts"
              @click="(event) => notificationPanel.toggle(event)"
            />
            <span
              v-if="activeAlerts.length > 0"
              class="absolute -top-1 -right-1 grid h-4 w-4 place-items-center rounded-full bg-red-600 text-[9px] font-bold text-white shadow"
            >
              {{ activeAlerts.length }}
            </span>
          </div>

          <Popover ref="notificationPanel" class="w-[320px]">
            <div class="grid gap-2">
              <div class="flex items-center justify-between border-b border-slate-200 pb-2">
                <strong class="text-xs font-bold text-brand-navy">Inventory Alerts ({{ activeAlerts.length }})</strong>
                <span class="text-[10px] text-brand-muted">Real-time</span>
              </div>
              <div v-if="activeAlerts.length === 0" class="py-4 text-center text-xs text-brand-muted">
                No active low stock or expiration alerts.
              </div>
              <div v-else class="grid gap-2 max-h-[220px] overflow-y-auto pr-1">
                <div
                  v-for="alert in activeAlerts.slice(0, 5)"
                  :key="alert.id"
                  class="rounded bg-slate-50 p-2 text-xs border border-slate-200"
                >
                  <strong class="block text-brand-navy">{{ alert.product_name || 'Item' }}</strong>
                  <div class="flex justify-between text-[11px] text-slate-600 mt-0.5">
                    <span>Qty: <strong class="text-red-600">{{ alert.quantity }}</strong></span>
                    <span v-if="alert.expiration_date" class="text-amber-700">Exp: {{ alert.expiration_date }}</span>
                  </div>
                </div>
              </div>
            </div>
          </Popover>

          <div v-if="auth.user" class="grid text-right text-xs">
            <span class="font-bold text-brand-navy">{{ auth.user.full_name }}</span>
            <span class="font-medium capitalize text-brand-muted text-[11px]">
              {{ auth.user.role.replace('_', ' ') }}
            </span>
          </div>

          <Button
            label="Sign out"
            icon="pi pi-sign-out"
            size="small"
            severity="secondary"
            outlined
            @click="signOut"
          />
        </div>
      </header>

      <main class="flex-1 p-[clamp(1rem,2.5vw,2rem)]">
        <RouterView />
      </main>

      <!-- Floating SpeedDial Quick Action Button -->
      <div class="fixed bottom-6 right-6 z-50">
        <SpeedDial
          :model="speedDialItems"
          direction="up"
          :radius="80"
          type="semi-circle"
          button-class="!bg-brand-navy !text-white !h-14 !w-14 shadow-xl border-2 border-brand-amber"
        />
      </div>
    </div>

    <!-- Modals -->
    <ReceiveFormDialog
      v-model:visible="receiveVisible"
      :locations="warehouseStore.locations"
      :products="catalogStore.products"
      @submitted="onMovementSuccess('Inventory received successfully!')"
    />

    <PickFormDialog
      v-model:visible="pickVisible"
      :locations="warehouseStore.locations"
      :products="catalogStore.products"
      @submitted="onMovementSuccess('FEFO stock pick executed!')"
    />

    <TransferFormDialog
      v-model:visible="transferVisible"
      :locations="warehouseStore.locations"
      :products="catalogStore.products"
      @submitted="onMovementSuccess('Stock transferred successfully!')"
    />

    <AdjustFormDialog
      v-model:visible="adjustVisible"
      :locations="warehouseStore.locations"
      :products="catalogStore.products"
      @submitted="onMovementSuccess('Cycle count adjustment recorded!')"
    />

    <BarcodeScannerModal
      v-model:visible="scannerVisible"
      @select="onBarcodeScanned"
    />
  </div>
</template>
