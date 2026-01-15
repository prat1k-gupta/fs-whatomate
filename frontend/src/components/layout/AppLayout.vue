<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useContactsStore } from '@/stores/contacts'
import { usersService, chatbotService } from '@/services/api'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Separator } from '@/components/ui/separator'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import {
  Popover,
  PopoverContent,
  PopoverTrigger
} from '@/components/ui/popover'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@/components/ui/alert-dialog'
import {
  LayoutDashboard,
  MessageSquare,
  Bot,
  FileText,
  Megaphone,
  Settings,
  LogOut,
  ChevronLeft,
  ChevronRight,
  Users,
  Workflow,
  Sparkles,
  Key,
  User,
  UserX,
  MessageSquareText,
  Sun,
  Moon,
  Monitor,
  Webhook,
  BarChart3,
  ShieldCheck,
  Zap,
  Instagram,
  Phone
} from 'lucide-vue-next'
import { useColorMode } from '@/composables/useColorMode'
import { toast } from 'vue-sonner'
import { getInitials } from '@/lib/utils'
import { wsService } from '@/services/websocket'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const contactsStore = useContactsStore()
const isCollapsed = ref(false)
const isUserMenuOpen = ref(false)
const isUpdatingAvailability = ref(false)
const isCheckingTransfers = ref(false)
const showAwayWarning = ref(false)
const awayWarningTransferCount = ref(0)
const { colorMode, isDark, setColorMode } = useColorMode()

const handleAvailabilityChange = async (checked: boolean) => {
  // If going away, check for assigned transfers first
  if (!checked) {
    isCheckingTransfers.value = true
    try {
      // Fetch current user's active transfers from API
      const response = await chatbotService.listTransfers({ status: 'active' })
      const data = response.data.data || response.data
      const transfers = data.transfers || []
      const userId = authStore.user?.id
      const myActiveTransfers = transfers.filter((t: any) => t.agent_id === userId)

      if (myActiveTransfers.length > 0) {
        awayWarningTransferCount.value = myActiveTransfers.length
        showAwayWarning.value = true
        return
      }
    } catch (error) {
      console.error('Failed to check transfers:', error)
      // Proceed anyway if check fails
    } finally {
      isCheckingTransfers.value = false
    }
  }

  await setAvailability(checked)
}

const confirmGoAway = async () => {
  showAwayWarning.value = false
  await setAvailability(false)
}

const setAvailability = async (checked: boolean) => {
  isUpdatingAvailability.value = true
  try {
    const response = await usersService.updateAvailability(checked)
    const data = response.data.data
    authStore.setAvailability(checked, data.break_started_at)

    if (checked) {
      toast.success('Available', {
        description: 'You are now available to receive transfers'
      })
    } else {
      const transfersReturned = data.transfers_to_queue || 0
      toast.success('Away', {
        description: transfersReturned > 0
          ? `${transfersReturned} transfer(s) returned to queue`
          : 'You will not receive new transfer assignments'
      })

      // Refresh contacts list if transfers were returned to queue
      if (transfersReturned > 0) {
        contactsStore.fetchContacts()
      }
    }
  } catch (error) {
    toast.error('Error', {
      description: 'Failed to update availability'
    })
  } finally {
    isUpdatingAvailability.value = false
  }
}

// Calculate break duration for display
const breakDuration = ref('')
let breakTimerInterval: ReturnType<typeof setInterval> | null = null

const updateBreakDuration = () => {
  if (!authStore.breakStartedAt) {
    breakDuration.value = ''
    return
  }
  const start = new Date(authStore.breakStartedAt)
  const now = new Date()
  const diffMs = now.getTime() - start.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const hours = Math.floor(diffMins / 60)
  const mins = diffMins % 60

  if (hours > 0) {
    breakDuration.value = `${hours}h ${mins}m`
  } else {
    breakDuration.value = `${mins}m`
  }
}

// Start/stop break timer based on availability
watch(() => authStore.isAvailable, (available) => {
  if (!available && authStore.breakStartedAt) {
    updateBreakDuration()
    breakTimerInterval = setInterval(updateBreakDuration, 60000) // Update every minute
  } else if (breakTimerInterval) {
    clearInterval(breakTimerInterval)
    breakTimerInterval = null
    breakDuration.value = ''
  }
}, { immediate: true })

// Restore break time on mount and connect WebSocket
onMounted(() => {
  authStore.restoreBreakTime()
  if (!authStore.isAvailable && authStore.breakStartedAt) {
    updateBreakDuration()
    breakTimerInterval = setInterval(updateBreakDuration, 60000)
  }

  // Connect WebSocket for real-time updates across all pages
  const token = localStorage.getItem('auth_token')
  if (token) {
    wsService.connect(token)
  }
})

onUnmounted(() => {
  if (breakTimerInterval) {
    clearInterval(breakTimerInterval)
  }
})

// Define all navigation items with role requirements
const allNavItems = [
  {
    name: 'Dashboard',
    path: '/',
    icon: LayoutDashboard,
    roles: ['admin', 'manager']
  },
  {
    name: 'Chat',
    path: '/chat',
    icon: MessageSquare,
    roles: ['admin', 'manager', 'agent']
  },
  {
    name: 'Chatbot',
    path: '/chatbot',
    icon: Bot,
    roles: ['admin', 'manager'],
    children: [
      { name: 'Overview', path: '/chatbot', icon: Bot },
      { name: 'Keywords', path: '/chatbot/keywords', icon: Key },
      { name: 'Flows', path: '/chatbot/flows', icon: Workflow },
      { name: 'AI Contexts', path: '/chatbot/ai', icon: Sparkles }
    ]
  },
  {
    name: 'Transfers',
    path: '/chatbot/transfers',
    icon: UserX,
    roles: ['admin', 'manager', 'agent']
  },
  {
    name: 'Agent Analytics',
    path: '/analytics/agents',
    icon: BarChart3,
    roles: ['admin', 'manager', 'agent']
  },
  {
    name: 'Templates',
    path: '/templates',
    icon: FileText,
    roles: ['admin', 'manager']
  },
  {
    name: 'Flows',
    path: '/flows',
    icon: Workflow,
    roles: ['admin', 'manager']
  },
  {
    name: 'Campaigns',
    path: '/campaigns',
    icon: Megaphone,
    roles: ['admin', 'manager']
  },
  {
    name: 'Settings',
    path: '/settings',
    icon: Settings,
    roles: ['admin', 'manager'],
    children: [
      { name: 'General', path: '/settings', icon: Settings },
      { name: 'Chatbot', path: '/settings/chatbot', icon: Bot },
      { name: 'WhatsApp Accounts', path: '/settings/accounts', icon: Phone },
      { name: 'Instagram Accounts', path: '/settings/instagram-accounts', icon: Instagram },
      { name: 'Canned Responses', path: '/settings/canned-responses', icon: MessageSquareText },
      { name: 'Teams', path: '/settings/teams', icon: Users },
      { name: 'Users', path: '/settings/users', icon: Users, roles: ['admin'] },
      { name: 'API Keys', path: '/settings/api-keys', icon: Key, roles: ['admin'] },
      { name: 'Webhooks', path: '/settings/webhooks', icon: Webhook, roles: ['admin'] },
      { name: 'Custom Actions', path: '/settings/custom-actions', icon: Zap, roles: ['admin'] },
      { name: 'SSO', path: '/settings/sso', icon: ShieldCheck, roles: ['admin'] }
    ]
  }
]

// Filter navigation based on user role
const navigation = computed(() => {
  const userRole = authStore.userRole || 'agent'

  return allNavItems
    .filter(item => item.roles.includes(userRole))
    .map(item => ({
      ...item,
      active: item.path === '/'
        ? route.name === 'dashboard'
        : item.path === '/chat'
          ? route.name === 'chat' || route.name === 'chat-conversation'
          : route.path.startsWith(item.path),
      children: item.children?.filter(
        child => !child.roles || child.roles.includes(userRole)
      )
    }))
})

const toggleSidebar = () => {
  isCollapsed.value = !isCollapsed.value
}

const handleLogout = async () => {
  await authStore.logout()
  router.push('/login')
}
</script>

<template>
  <div class="flex h-screen bg-background">
    <!-- Sidebar -->
    <aside
      :class="[
        'flex flex-col border-r bg-card transition-all duration-300',
        isCollapsed ? 'w-16' : 'w-64'
      ]"
    >
      <!-- Logo -->
      <div class="flex h-12 items-center justify-between px-3 border-b">
        <RouterLink to="/" class="flex items-center">
          <svg v-if="!isCollapsed" viewBox="0 0 555 103" fill="none" xmlns="http://www.w3.org/2000/svg" class="w-32 h-8"><g><path d="M104.485 43.6502L110.811 36.9628V65.0225L106.473 61.4981H133.719V72.3424H106.473L110.811 68.5469V99.4531H96.8035V32.8058H137.741V43.6502H104.485ZM170.052 32.8058C177.071 32.8058 182.252 34.4927 185.596 37.8665C188.939 41.2403 190.611 45.9094 190.611 51.8737C190.611 55.8199 190.009 59.1334 188.804 61.8144C187.599 64.4953 185.927 66.6491 183.788 68.2758C181.68 69.8723 179.24 71.017 176.468 71.7098C173.697 72.4026 170.73 72.749 167.567 72.749H152.656L156.677 68.4113V99.4531H142.941V32.8058H170.052ZM167.702 62.221C169.359 62.221 170.82 61.8595 172.085 61.1366C173.381 60.3835 174.375 59.284 175.067 57.8381C175.79 56.3621 176.152 54.5547 176.152 52.416C176.152 49.5543 175.339 47.3252 173.712 45.7286C172.085 44.102 169.736 43.2887 166.663 43.2887H152.972L156.677 38.9509V66.378L153.153 62.221H167.702ZM164.946 68.095H178.004L190.25 99.4531H175.881L164.946 68.095ZM203.558 43.6502L209.884 36.9628V64.5706L206.721 60.5944H231.301V71.4387H206.721L209.884 67.4624V95.2961L203.558 88.6088H236.317V99.4531H195.786V32.8058H236.317V43.6502H203.558ZM250.578 43.6502L256.904 36.9628V64.5706L253.741 60.5944H278.321V71.4387H253.741L256.904 67.4624V95.2961L250.578 88.6088H283.337V99.4531H242.806V32.8058H283.337V43.6502H250.578ZM312.419 100.538C307.599 100.538 303.352 99.7694 299.677 98.2331C296.032 96.6969 293.17 94.4075 291.092 91.3651C289.043 88.2925 288.019 84.482 288.019 79.9334C288.019 79.4514 288.019 79.0146 288.019 78.623C288.019 78.2013 288.019 77.7495 288.019 77.2675H302.071C302.071 77.7193 302.071 78.126 302.071 78.4875C302.071 78.8188 302.071 79.2104 302.071 79.6623C302.071 82.7649 302.945 85.1447 304.692 86.8014C306.439 88.4582 309.03 89.2866 312.464 89.2866C315.928 89.2866 318.594 88.7444 320.462 87.6599C322.329 86.5454 323.263 84.7229 323.263 82.1926C323.263 80.4153 322.555 78.8489 321.139 77.4934C319.754 76.1379 317.856 74.933 315.446 73.8786C313.066 72.7942 310.34 71.7851 307.268 70.8513C303.924 69.797 300.821 68.4113 297.96 66.6943C295.098 64.9773 292.794 62.7934 291.046 60.1425C289.329 57.4917 288.471 54.2233 288.471 50.3375C288.471 46.4215 289.51 43.0929 291.589 40.3517C293.667 37.5804 296.499 35.4717 300.083 34.0258C303.668 32.5498 307.72 31.8118 312.238 31.8118C316.967 31.8118 321.139 32.5498 324.754 34.0258C328.399 35.4717 331.246 37.6557 333.294 40.5776C335.373 43.4694 336.412 47.0842 336.412 51.4219C336.412 51.9039 336.412 52.3557 336.412 52.7774C336.412 53.169 336.412 53.6209 336.412 54.133H322.405C322.405 53.8016 322.405 53.4552 322.405 53.0937C322.405 52.7323 322.405 52.3858 322.405 52.0545C322.405 49.1325 321.591 46.8884 319.965 45.322C318.368 43.7556 315.853 42.9724 312.419 42.9724C309.105 42.9724 306.53 43.5748 304.692 44.7798C302.885 45.9847 301.981 47.8373 301.981 50.3375C301.981 52.1449 302.629 53.6811 303.924 54.9463C305.249 56.2115 307.057 57.3411 309.346 58.3351C311.636 59.3292 314.241 60.3233 317.163 61.3173C321.139 62.703 324.604 64.2393 327.556 65.9262C330.538 67.5829 332.842 69.6765 334.469 72.2068C336.095 74.707 336.909 77.9001 336.909 81.7859C336.909 85.7622 335.885 89.151 333.836 91.9525C331.788 94.7539 328.911 96.8927 325.206 98.3687C321.531 99.8146 317.269 100.538 312.419 100.538ZM386.752 43.6502H364.747L370.35 36.9628V99.4531H356.388V36.9628L362.171 43.6502H340.031V32.8058H386.752V43.6502ZM381.323 99.4531L397.951 32.7155H420.137L436.674 99.4531H422.306L409.021 41.2554H409.112L395.692 99.4531H381.323ZM393.252 84V73.0653H424.881V84H393.252ZM453.622 99.4531H439.931V32.8058H461.71L484.031 94.3925L481.682 94.9798V32.8058H495.327V99.4531H473.413L451.092 38.0473L453.622 37.4599V99.4531ZM501.866 99.4531V32.8058H523.419C527.877 32.8058 531.929 33.6192 535.573 35.2458C539.248 36.8423 542.396 39.1166 545.017 42.0687C547.668 44.9906 549.701 48.4849 551.117 52.5515C552.533 56.6181 553.241 61.1065 553.241 66.0165C553.241 70.9266 552.533 75.43 551.117 79.5267C549.701 83.6235 547.668 87.1629 545.017 90.1451C542.396 93.0972 539.248 95.3865 535.573 97.0132C531.929 98.6398 527.877 99.4531 523.419 99.4531H501.866ZM515.963 95.2961L509.638 88.6088H521.611C524.865 88.6088 527.802 87.7051 530.422 85.8977C533.073 84.0602 535.167 81.4546 536.703 78.0808C538.239 74.6769 539.007 70.6555 539.007 66.0165C539.007 61.3475 538.239 57.3561 536.703 54.0426C535.167 50.6989 533.073 48.1385 530.422 46.3612C527.802 44.5538 524.865 43.6502 521.611 43.6502H509.638L515.963 36.9628V95.2961Z" fill="white"></path><path d="M69.8052 54.1819V94.3842C69.7538 96.1843 69.7565 97.3416 69.7565 97.3416H43.3398V54.1819M69.8052 54.1819H43.3398M69.8052 54.1819H73.7703V43.4453L43.3398 43.4453V54.1819" stroke="white" stroke-width="7" stroke-miterlimit="16" stroke-linecap="round"></path><path d="M7.65428 54.1819V94.3842C7.70567 96.1843 7.70296 97.3416 7.70296 97.3416H32.2754V54.1819M7.65428 54.1819H32.2754M7.65428 54.1819H3.68922V43.4453L32.2754 43.4453V54.1819" stroke="white" stroke-width="7" stroke-miterlimit="16" stroke-linecap="round"></path><path d="M3.68945 33.6827C8.22718 33.6827 26.7948 33.6827 31.2307 33.6827L29.6309 31.8631C24.9526 26.0405 15.9363 20.2184 16.3726 10.3922C16.5859 5.58911 23.4992 0.0570863 30.553 7.04422C36.1961 12.6339 37.1221 22.9108 36.8797 27.3505C39.1825 20.8971 45.737 9.08208 53.5326 13.449C63.2771 18.9077 51.3772 29.1701 48.9775 30.1891C47.0577 31.2371 45.1728 32.7122 44.0093 33.3187H73.7717" stroke="white" stroke-width="7" stroke-linecap="square" stroke-linejoin="round"></path></g><defs><clipPath id="clip0_432_613"><rect width="555" height="103" fill="white"></rect></clipPath></defs></svg>
          <svg v-else viewBox="0 0 127 128" fill="none" xmlns="http://www.w3.org/2000/svg" class="w-10 h-10"><g><path d="M94.8052 62.1819V102.384C94.7538 104.184 94.7565 105.342 94.7565 105.342H68.3398V62.1819M94.8052 62.1819H68.3398M94.8052 62.1819H98.7703V51.4453L68.3398 51.4453V62.1819" stroke="white" stroke-width="7" stroke-miterlimit="16" stroke-linecap="round"></path><path d="M32.6543 62.1819V102.384C32.7057 104.184 32.703 105.342 32.703 105.342H57.2754V62.1819M32.6543 62.1819H57.2754M32.6543 62.1819H28.6892V51.4453L57.2754 51.4453V62.1819" stroke="white" stroke-width="7" stroke-miterlimit="16" stroke-linecap="round"></path><path d="M28.6895 41.6827C33.2272 41.6827 51.7948 41.6827 56.2307 41.6827L54.6309 39.8631C49.9526 34.0405 40.9363 28.2184 41.3726 18.3922C41.5859 13.5891 48.4992 8.05709 55.553 15.0442C61.1961 20.6339 62.1221 30.9108 61.8797 35.3505C64.1825 28.8971 70.737 17.0821 78.5326 21.449C88.2771 26.9077 76.3772 37.1701 73.9775 38.1891C72.0577 39.2371 70.1728 40.7122 69.0093 41.3187H98.7717" stroke="white" stroke-width="7" stroke-linecap="square" stroke-linejoin="round"></path></g></svg>
        </RouterLink>
        <Button
          variant="ghost"
          size="icon"
          class="h-7 w-7"
          @click="toggleSidebar"
        >
          <ChevronLeft v-if="!isCollapsed" class="h-3.5 w-3.5" />
          <ChevronRight v-else class="h-3.5 w-3.5" />
        </Button>
      </div>

      <!-- Navigation -->
      <ScrollArea class="flex-1 py-2">
        <nav class="space-y-0.5 px-2">
          <template v-for="item in navigation" :key="item.path">
            <RouterLink
              :to="item.path"
              :class="[
                'flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-[13px] font-medium transition-colors',
                item.active
                  ? 'bg-primary/10 text-primary'
                  : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
                isCollapsed && 'justify-center px-2'
              ]"
            >
              <component :is="item.icon" class="h-4 w-4 shrink-0" />
              <span v-if="!isCollapsed">{{ item.name }}</span>
            </RouterLink>

            <!-- Submenu items -->
            <template v-if="item.children && item.active && !isCollapsed">
              <RouterLink
                v-for="child in item.children"
                :key="child.path"
                :to="child.path"
                :class="[
                  'flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-[13px] font-medium transition-colors ml-4',
                  route.path === child.path
                    ? 'bg-primary/10 text-primary'
                    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                ]"
              >
                <component :is="child.icon" class="h-3.5 w-3.5 shrink-0" />
                <span>{{ child.name }}</span>
              </RouterLink>
            </template>
          </template>
        </nav>
      </ScrollArea>

      <!-- User section -->
      <div class="border-t p-2">
        <Popover v-model:open="isUserMenuOpen">
          <PopoverTrigger as-child>
            <Button
              variant="ghost"
              :class="[
                'flex items-center w-full h-auto px-2 py-1.5 gap-2',
                isCollapsed && 'justify-center'
              ]"
            >
              <Avatar class="h-7 w-7">
                <AvatarImage :src="undefined" />
                <AvatarFallback class="text-xs">
                  {{ getInitials(authStore.user?.full_name || 'U') }}
                </AvatarFallback>
              </Avatar>
              <div v-if="!isCollapsed" class="flex flex-col items-start text-left">
                <span class="text-[13px] font-medium truncate max-w-[140px]">
                  {{ authStore.user?.full_name }}
                </span>
                <span class="text-[11px] text-muted-foreground truncate max-w-[140px]">
                  {{ authStore.user?.email }}
                </span>
              </div>
            </Button>
          </PopoverTrigger>
          <PopoverContent side="top" align="start" class="w-52 p-1.5">
            <div class="text-xs font-medium px-2 py-1 text-muted-foreground">My Account</div>
            <Separator class="my-1" />
            <!-- Availability Toggle -->
            <div class="flex items-center justify-between px-2 py-1.5">
              <div class="flex items-center gap-2">
                <span class="text-[13px]">Status</span>
                <Badge :variant="authStore.isAvailable ? 'default' : 'secondary'" class="text-[10px] px-1.5 py-0">
                  {{ authStore.isAvailable ? 'Available' : 'Away' }}
                </Badge>
                <span v-if="!authStore.isAvailable && breakDuration" class="text-[10px] text-muted-foreground">
                  {{ breakDuration }}
                </span>
              </div>
              <Switch
                :checked="authStore.isAvailable"
                :disabled="isUpdatingAvailability || isCheckingTransfers"
                @update:checked="handleAvailabilityChange"
              />
            </div>
            <Separator class="my-1" />
            <RouterLink to="/profile">
              <Button
                variant="ghost"
                class="w-full justify-start px-2 py-1 h-auto text-[13px] font-normal"
                @click="isUserMenuOpen = false"
              >
                <User class="mr-2 h-3.5 w-3.5" />
                <span>Profile</span>
              </Button>
            </RouterLink>
            <Separator class="my-1" />
            <div class="text-xs font-medium px-2 py-1 text-muted-foreground">Theme</div>
            <div class="flex gap-0.5 px-1.5 py-1">
              <Button
                variant="ghost"
                size="icon"
                class="h-7 w-7"
                :class="colorMode === 'light' && 'bg-accent'"
                @click="setColorMode('light')"
              >
                <Sun class="h-3.5 w-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                class="h-7 w-7"
                :class="colorMode === 'dark' && 'bg-accent'"
                @click="setColorMode('dark')"
              >
                <Moon class="h-3.5 w-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                class="h-7 w-7"
                :class="colorMode === 'system' && 'bg-accent'"
                @click="setColorMode('system')"
              >
                <Monitor class="h-3.5 w-3.5" />
              </Button>
            </div>
            <Separator class="my-1" />
            <Button
              variant="ghost"
              class="w-full justify-start px-2 py-1 h-auto text-[13px] font-normal"
              @click="handleLogout"
            >
              <LogOut class="mr-2 h-3.5 w-3.5" />
              <span>Log out</span>
            </Button>
          </PopoverContent>
        </Popover>
      </div>
    </aside>

    <!-- Main content -->
    <main class="flex-1 overflow-hidden">
      <RouterView />
    </main>

    <!-- Away Warning Dialog -->
    <AlertDialog :open="showAwayWarning">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Active Transfers Will Be Returned to Queue</AlertDialogTitle>
          <AlertDialogDescription>
            You have {{ awayWarningTransferCount }} active transfer(s) assigned to you.
            Setting your status to "Away" will return them to the queue for other agents to pick up.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <Button variant="outline" @click="showAwayWarning = false">Cancel</Button>
          <Button @click="confirmGoAway" :disabled="isUpdatingAvailability">Go Away</Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
