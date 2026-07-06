import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/im/home',
    },
    {
      path: '/im/enter',
      name: 'im-enter',
      component: () => import('@/views/im/enter.vue'),
    },
    {
      path: '/im/login',
      name: 'im-login',
      component: () => import('@/views/im/login.vue'),
      meta: { guest: true },
    },
    {
      path: '/im',
      component: () => import('@/views/im/index.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          redirect: { name: 'im-home' },
        },
        {
          path: 'home',
          name: 'im-home',
          component: () => import('@/views/im/home.vue'),
        },
        {
          path: 'chat',
          name: 'im-chat-index',
          component: () => import('@/views/im/chat.vue'),
          props: { conversationId: '' },
        },
        {
          path: 'chat/:conversationId',
          name: 'im-chat',
          component: () => import('@/views/im/chat.vue'),
          props: true,
        },
        {
          path: 'video-player',
          name: 'im-video-player',
          component: () => import('@/views/im/video-player.vue'),
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/im/home',
    },
  ],
})

router.beforeEach((to) => {
  const userStore = useUserStore()

  if (to.meta.requiresAuth && !userStore.token) {
    return {
      name: 'im-login',
      query: { redirect: to.fullPath },
    }
  }

  if (to.meta.guest && userStore.token) {
    const redirect = typeof to.query.redirect === 'string' ? to.query.redirect : '/im/home'
    return redirect
  }
})

export default router
