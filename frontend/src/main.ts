import Aura from '@primeuix/themes/aura'
import { createPinia } from 'pinia'
import ConfirmationService from 'primevue/confirmationservice'
import PrimeVue from 'primevue/config'
import ToastService from 'primevue/toastservice'
import { createApp } from 'vue'

import App from './App.vue'
import router from './router'
import './assets/main.css'
import 'primeicons/primeicons.css'

const pinia = createPinia()

createApp(App)
  .use(pinia)
  .use(PrimeVue, {
    theme: {
      preset: Aura,
      options: {
        darkModeSelector: '.bwims-dark',
      },
    },
  })
  .use(ToastService)
  .use(ConfirmationService)
  .use(router)
  .mount('#app')
