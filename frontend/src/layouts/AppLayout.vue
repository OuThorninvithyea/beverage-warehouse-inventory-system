<script setup lang="ts">
import {
  ArrowLeftRight,
  Bell,
  Boxes,
  Building2,
  Camera,
  ClipboardList,
  Database,
  LayoutGrid,
  LogOut,
  Menu,
  Moon,
  Plus,
  ScanLine,
  SlidersHorizontal,
  Sun,
  Tags,
  Upload,
  Download,
  Users,
} from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'
import { toast } from 'vue-sonner'
import { RouterLink, RouterView, useRouter } from 'vue-router'

import AdjustFormDialog from '@/components/AdjustFormDialog.vue'
import BarcodeScannerModal from '@/components/BarcodeScannerModal.vue'
import PickFormDialog from '@/components/PickFormDialog.vue'
import ReceiveFormDialog from '@/components/ReceiveFormDialog.vue'
import TransferFormDialog from '@/components/TransferFormDialog.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Separator } from '@/components/ui/separator'
import { Toaster } from '@/components/ui/sonner'
import type { Lot } from '@/api/inventory'
import { useAuthStore } from '@/stores/auth'
import { useCatalogStore } from '@/stores/catalog'
import { useInventoryStore } from '@/stores/inventory'
import { useWarehousesStore } from '@/stores/warehouses'

const auth = useAuthStore()
const router = useRouter()
const catalogStore = useCatalogStore()
const warehouseStore = useWarehousesStore()
const inventoryStore = useInventoryStore()

const mobileMenuOpen = ref(false)
const isDarkMode = ref(false)
const lotsByProduct = ref<Map<string, Lot[]>>(new Map())

const receiveVisible = ref(false)
const pickVisible = ref(false)
const transferVisible = ref(false)
const adjustVisible = ref(false)
const scannerVisible = ref(false)

const isAdmin = computed(() => auth.user?.role === 'admin')
const isManager = computed(
  () => auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager',
)
const canReceiveOrPick = computed(
  () =>
    auth.user?.role === 'admin' ||
    auth.user?.role === 'warehouse_manager' ||
    auth.user?.role === 'picker',
)

const navSections = computed(() => {
  const sections: Array<{
    label: string | null
    items: Array<{ to: string; label: string; icon: typeof LayoutGrid; exact: boolean }>
  }> = [
    {
      label: null,
      items: [{ to: '/', label: 'Dashboard', icon: LayoutGrid, exact: true }],
    },
    {
      label: 'Product Catalog',
      items: [
        { to: '/products', label: 'Products', icon: Boxes, exact: false },
        { to: '/categories', label: 'Categories', icon: Tags, exact: false },
      ],
    },
    {
      label: 'Warehouses',
      items: [{ to: '/warehouses', label: 'Warehouses', icon: Building2, exact: false }],
    },
    {
      label: 'Inventory Ops',
      items: [
        { to: '/inventory', label: 'Inventory', icon: Database, exact: false },
        { to: '/movements', label: 'Movement History', icon: ClipboardList, exact: false },
      ],
    },
  ]

  if (isAdmin.value) {
    sections.push({
      label: 'Admin & Tools',
      items: [
        { to: '/users', label: 'User Management', icon: Users, exact: false },
        { to: '/barcode-test', label: 'Barcode Tools', icon: Camera, exact: false },
      ],
    })
  }

  return sections
})

const userInitials = computed(() => {
  const name = auth.user?.full_name?.trim()
  if (!name) return 'BW'
  return name
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? '')
    .join('')
})

const activeAlerts = computed(() => {
  return inventoryStore.balances
    .map((b) => {
      const product = catalogStore.products.find((p) => p.id === b.product_id)
      const lot = b.lot_id
        ? (lotsByProduct.value.get(b.product_id) ?? []).find((l) => l.id === b.lot_id)
        : undefined
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
        const diffDays = Math.ceil(
          (new Date(b.expiration_date).getTime() - new Date().getTime()) / (1000 * 3600 * 24),
        )
        isExp = diffDays <= 30
      }
      return isLow || isExp
    })
})

onMounted(() => {
  const storedTheme = localStorage.getItem('bwims_theme')
  if (storedTheme === 'dark') {
    isDarkMode.value = true
    document.documentElement.classList.add('dark', 'bwims-dark')
  }

  // Preload data the quick-action dialogs (Receive/Pick/Transfer/Adjust) need,
  // so they aren't empty if the user hasn't visited Inventory/Movements yet.
  void catalogStore.fetchProducts()
  void warehouseStore.fetchAllLocations()
  inventoryStore.fetchBalances().then(async () => {
    const productIds = new Set(
      inventoryStore.balances.filter((b) => b.lot_id).map((b) => b.product_id),
    )
    for (const productId of productIds) {
      const lots = await inventoryStore.fetchLots(productId)
      lotsByProduct.value.set(productId, lots)
    }
  })
})

function toggleTheme() {
  isDarkMode.value = !isDarkMode.value
  document.documentElement.classList.toggle('dark', isDarkMode.value)
  document.documentElement.classList.toggle('bwims-dark', isDarkMode.value)
  localStorage.setItem('bwims_theme', isDarkMode.value ? 'dark' : 'light')
}

const quickActions = computed(() => {
  const actions: { label: string; icon: typeof Upload; run: () => void }[] = []
  if (canReceiveOrPick.value) {
    actions.push(
      { label: 'Receive Stock', icon: Download, run: () => (receiveVisible.value = true) },
      { label: 'FEFO Pick', icon: Upload, run: () => (pickVisible.value = true) },
    )
  }
  if (isManager.value) {
    actions.push(
      { label: 'Transfer', icon: ArrowLeftRight, run: () => (transferVisible.value = true) },
      { label: 'Adjust', icon: SlidersHorizontal, run: () => (adjustVisible.value = true) },
    )
  }
  actions.push({ label: 'Scan Barcode', icon: ScanLine, run: () => (scannerVisible.value = true) })
  return actions
})

async function signOut() {
  await auth.signOut()
  await router.push({ name: 'login' })
}

function closeMobileMenu() {
  mobileMenuOpen.value = false
}

function onBarcodeScanned(code: string) {
  toast.info('Barcode Scanned', { description: `Code: ${code}` })
  void router.push({ name: 'products', query: { search: code } })
}

function onMovementSuccess(msg: string) {
  toast.success('Operation Complete', { description: msg })
  void inventoryStore.fetchBalances()
  void inventoryStore.fetchMovements()
}
</script>

<template>
  <div class="grid min-h-screen grid-cols-[264px_minmax(0,1fr)] bg-muted/40 max-[900px]:grid-cols-1">

    <!-- Mobile top bar -->
    <div class="hidden items-center justify-between border-b bg-sidebar px-4 py-3 text-sidebar-foreground max-[900px]:flex">
      <div class="flex items-center gap-3">
        <span class="grid size-8 place-items-center rounded-lg bg-brand-amber text-xs font-extrabold text-brand-navy">BW</span>
        <strong class="text-sm font-semibold">BWIMS</strong>
      </div>
      <Button variant="ghost" size="icon" aria-label="Toggle navigation" @click="mobileMenuOpen = !mobileMenuOpen">
        <Menu class="size-5" />
      </Button>
    </div>

    <!-- Sidebar -->
    <aside
      class="flex flex-col gap-6 border-r bg-sidebar p-4 text-sidebar-foreground max-[900px]:border-r-0 max-[900px]:border-b"
      :class="{ 'max-[900px]:hidden': !mobileMenuOpen }"
    >
      <div class="flex items-center gap-3 px-2 pt-2">
        <span class="grid size-10 place-items-center rounded-xl bg-brand-amber text-base font-extrabold text-brand-navy shadow-sm">
          BW
        </span>
        <div class="grid leading-tight">
          <strong class="text-sm font-semibold tracking-tight">BWIMS</strong>
          <small class="text-xs text-muted-foreground">Beverage Warehouse Control</small>
        </div>
      </div>

        <nav aria-label="Primary navigation" class="grid gap-4">
          <div v-for="section in navSections" :key="section.label ?? 'dashboard'" class="grid gap-1">
            <p v-if="section.label" class="px-3 pb-1 text-[10px] font-bold uppercase tracking-[0.16em] text-muted-foreground">
              {{ section.label }}
            </p>
            <RouterLink
              v-for="item in section.items"
              :key="item.to"
              :to="item.to"
              class="group flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
              :class="
                item.exact
                  ? '[&.router-link-exact-active]:bg-primary [&.router-link-exact-active]:text-primary-foreground'
                  : '[&.router-link-active]:bg-primary [&.router-link-active]:text-primary-foreground'
              "
              @click="closeMobileMenu"
            >
              <component :is="item.icon" class="size-4" />
              <span>{{ item.label }}</span>
            </RouterLink>
          </div>
        </nav>

      <div class="mt-auto grid gap-1 rounded-xl border bg-card p-3 text-xs max-[900px]:hidden">
        <div class="flex items-center justify-between font-semibold text-muted-foreground">
          <span>System Status</span>
          <span class="inline-flex items-center gap-1 text-[10px] text-emerald-600">
            <span class="size-1.5 animate-pulse rounded-full bg-emerald-500" /> Live
          </span>
        </div>
        <p class="mt-1 text-[11px] leading-relaxed text-muted-foreground">
          Multi-warehouse stock control, FEFO picking, and audit trail active.
        </p>
      </div>
    </aside>

    <!-- Main -->
    <div class="relative flex min-w-0 flex-col">
      <header class="sticky top-0 z-30 flex min-h-[68px] items-center justify-between gap-4 border-b bg-background/85 px-6 py-3 backdrop-blur max-[520px]:px-4">
        <div class="grid gap-0.5">
          <small class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Beverage Distributor</small>
          <strong class="text-sm font-semibold max-[520px]:text-xs">Inventory Management System</strong>
        </div>

        <div class="flex items-center gap-2">
          <Button variant="ghost" size="icon" :title="isDarkMode ? 'Switch to light mode' : 'Switch to dark mode'" @click="toggleTheme">
            <Sun v-if="isDarkMode" class="size-4" />
            <Moon v-else class="size-4" />
          </Button>

          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <Button variant="ghost" size="icon" class="relative" title="Notifications & alerts">
                <Bell class="size-4" />
                <span
                  v-if="activeAlerts.length > 0"
                  class="absolute -right-0.5 -top-0.5 grid size-4 place-items-center rounded-full bg-destructive text-[9px] font-bold text-white"
                >
                  {{ activeAlerts.length }}
                </span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" class="w-80">
              <DropdownMenuLabel class="flex items-center justify-between">
                <span>Inventory Alerts ({{ activeAlerts.length }})</span>
                <span class="text-[10px] font-normal text-muted-foreground">Real-time</span>
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <div v-if="activeAlerts.length === 0" class="px-2 py-6 text-center text-xs text-muted-foreground">
                No active low stock or expiration alerts.
              </div>
              <div v-else class="max-h-[240px] overflow-y-auto">
                <DropdownMenuItem
                  v-for="alert in activeAlerts.slice(0, 6)"
                  :key="alert.id"
                  class="flex-col items-start gap-1"
                >
                  <strong class="text-xs">{{ alert.product_name }}</strong>
                  <div class="flex w-full justify-between text-[11px] text-muted-foreground">
                    <span>Qty: <strong class="text-destructive">{{ alert.quantity }}</strong></span>
                    <span v-if="alert.expiration_date" class="text-amber-600">Exp: {{ alert.expiration_date }}</span>
                  </div>
                </DropdownMenuItem>
              </div>
            </DropdownMenuContent>
          </DropdownMenu>

          <Separator orientation="vertical" class="mx-1 h-8 max-[520px]:hidden" />

          <DropdownMenu v-if="auth.user">
            <DropdownMenuTrigger as-child>
              <Button variant="ghost" class="h-10 gap-2 px-2">
                <span class="grid size-8 place-items-center rounded-full bg-primary/10 text-xs font-bold text-primary">
                  {{ userInitials }}
                </span>
                <span class="grid text-left leading-tight max-[640px]:hidden">
                  <span class="text-xs font-semibold">{{ auth.user.full_name }}</span>
                  <span class="text-[11px] capitalize text-muted-foreground">{{ auth.user.role.replace('_', ' ') }}</span>
                </span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" class="w-56">
              <DropdownMenuLabel class="grid gap-1">
                <span class="text-sm">{{ auth.user.full_name }}</span>
                <Badge variant="secondary" class="w-fit capitalize">{{ auth.user.role.replace('_', ' ') }}</Badge>
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem variant="destructive" @click="signOut">
                <LogOut class="size-4" />
                Sign out
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </header>

      <main class="flex-1 p-[clamp(1rem,2.5vw,2rem)]">
        <RouterView />
      </main>

      <!-- Quick actions -->
      <div class="fixed bottom-6 right-6 z-50">
        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button size="icon" class="size-14 rounded-full shadow-xl" aria-label="Quick actions">
              <Plus class="size-6" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" side="top" class="w-48">
            <DropdownMenuLabel>Quick actions</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem v-for="action in quickActions" :key="action.label" @click="action.run()">
              <component :is="action.icon" class="size-4" />
              {{ action.label }}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
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

    <BarcodeScannerModal v-model:visible="scannerVisible" @select="onBarcodeScanned" />
  </div>

  <Toaster position="top-right" rich-colors close-button />
</template>
