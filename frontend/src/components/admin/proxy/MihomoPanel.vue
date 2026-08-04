<template>
  <section
    ref="rootEl"
    class="min-w-0"
  >
    <header class="flex flex-wrap items-start justify-between gap-3 border-b border-gray-200 py-4 dark:border-dark-700">
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-2">
          <h2 class="font-semibold text-gray-900 dark:text-white">Mihomo 线路工作台</h2>
          <span :class="['badge', runtimeHealthy ? 'badge-success' : 'badge-danger']">
            {{ runtimeHealthy ? '运行正常' : loading ? '加载中' : '需要检查' }}
          </span>
          <span class="badge badge-gray">{{ subscriptions.length }} 个订阅</span>
          <span class="badge badge-gray">{{ routes.length }} 条线路</span>
        </div>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          订阅、节点、出口线路和托管 IP 在这里统一管理。{{ changePolicyHint }}
        </p>
      </div>
      <div class="flex flex-wrap gap-2">
        <button type="button" class="btn btn-secondary min-h-11" :disabled="loading" title="刷新工作台" @click="load">
          <Icon name="refresh" size="sm" :class="['mr-1.5', loading ? 'animate-spin' : '']" />刷新
        </button>
        <button type="button" class="btn btn-primary min-h-11" @click="activeTab = 'routes'; openRouteForm()">
          <Icon name="plus" size="sm" class="mr-1.5" />新建线路
        </button>
      </div>
    </header>

    <div v-if="loadError" class="my-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/20 dark:text-red-300">
      {{ loadError }}
    </div>

    <div v-if="importPreview?.available" class="border-y border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/40">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0 flex-1">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">导入现有 Mihomo 配置</h3>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ importPreview.subscription_host }} · {{ importPreview.node_count }} 个节点 · {{ importPreview.affected_account_count }} 个绑定账号
          </p>
        </div>
        <button type="button" class="btn btn-primary min-h-11" @click="confirmAction = { type: 'legacy-import' }">
          <Icon name="upload" size="sm" class="mr-1.5" />确认导入
        </button>
      </div>
      <dl class="mt-3 grid min-w-0 gap-2 sm:grid-cols-3">
        <div v-for="route in importPreview.routes" :key="route.listener_port" class="min-w-0 border-l-2 border-gray-300 pl-3 dark:border-dark-600">
          <dt class="truncate text-xs font-medium text-gray-800 dark:text-gray-100" :title="route.name">{{ route.name }}</dt>
          <dd class="mt-1 text-xs text-gray-500 dark:text-gray-400">端口 {{ route.listener_port }} · 代理 #{{ route.proxy_id || '-' }} · {{ route.account_count }} 个账号</dd>
        </div>
      </dl>
    </div>

    <template v-if="workbench">
      <nav class="overflow-x-auto border-b border-gray-200 dark:border-dark-700" aria-label="Mihomo 工作台视图">
        <div class="flex min-w-max px-2">
          <button
            v-for="tab in tabs"
            :key="tab.value"
            type="button"
            class="min-h-11 border-b-2 px-4 text-sm font-medium"
            :class="activeTab === tab.value ? 'border-primary-500 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100'"
            @click="activeTab = tab.value"
          >
            {{ tab.label }} <span class="ml-1 text-xs">{{ tabCount(tab.value) }}</span>
          </button>
        </div>
      </nav>

      <div class="min-w-0 py-4">
        <div v-if="activeTab === 'subscriptions'" class="min-w-0 space-y-3">
          <div class="flex flex-wrap items-center gap-2">
            <div class="relative min-w-0 flex-1 sm:max-w-sm">
              <Icon name="search" size="sm" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
              <input v-model.trim="subscriptionQuery" type="search" class="input min-w-0 pl-9" placeholder="搜索订阅名称或来源" aria-label="搜索 Mihomo 订阅" />
            </div>
            <button type="button" class="btn btn-primary min-h-11" @click="openSubscriptionForm()">
              <Icon name="plus" size="sm" class="mr-1.5" />添加订阅
            </button>
          </div>

          <div class="grid min-w-0 gap-3 xl:grid-cols-2">
            <article v-for="subscription in filteredSubscriptions" :key="subscription.id" class="min-w-0 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
              <div class="flex min-w-0 items-start justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <div class="flex min-w-0 flex-wrap items-center gap-2">
                    <h3 class="truncate text-sm font-semibold text-gray-900 dark:text-white" :title="subscription.name">{{ subscription.name }}</h3>
                    <span :class="['badge', subscriptionBadge(subscription)]">{{ subscriptionStatusLabel(subscription) }}</span>
                  </div>
                  <p class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400" :title="subscription.source_host || subscription.masked_url || '-'">
                    {{ subscription.source_host || subscription.masked_url || '-' }}
                  </p>
                </div>
                <span class="badge badge-gray shrink-0">{{ subscription.alive_count }}/{{ subscription.node_count }} 可用</span>
              </div>
              <dl class="mt-3 grid grid-cols-2 gap-3 text-xs sm:grid-cols-4">
                <div><dt class="text-gray-500">刷新周期</dt><dd class="mt-1 font-medium text-gray-800 dark:text-gray-100">{{ subscription.refresh_interval_minutes }} 分钟</dd></div>
                <div><dt class="text-gray-500">流量</dt><dd class="mt-1 font-medium text-gray-800 dark:text-gray-100">{{ quotaLabel(subscription) }}</dd></div>
                <div><dt class="text-gray-500">到期</dt><dd class="mt-1 font-medium text-gray-800 dark:text-gray-100">{{ shortTime(subscription.expires_at) }}</dd></div>
                <div><dt class="text-gray-500">最近刷新</dt><dd class="mt-1 font-medium text-gray-800 dark:text-gray-100">{{ shortTime(subscription.last_refreshed_at) }}</dd></div>
              </dl>
              <p v-if="subscription.last_error" class="mt-3 break-words text-xs text-red-600 dark:text-red-400">{{ subscription.last_error }}</p>
              <div class="mt-3 flex flex-wrap gap-2">
                <button type="button" class="btn btn-secondary min-h-11 px-3" @click="confirmSubscriptionRefresh(subscription)"><Icon name="refresh" size="sm" class="mr-1.5" />刷新</button>
                <button type="button" class="btn btn-secondary min-h-11 px-3" @click="openSubscriptionForm(subscription)"><Icon name="edit" size="sm" class="mr-1.5" />编辑</button>
                <button type="button" class="btn btn-secondary min-h-11 px-3" @click="confirmSubscriptionToggle(subscription)">{{ subscription.enabled ? '停用' : '启用' }}</button>
                <button type="button" class="btn btn-secondary min-h-11 px-3 text-red-600" @click="confirmDeleteSubscription(subscription)"><Icon name="trash" size="sm" class="mr-1.5" />删除</button>
              </div>
            </article>
          </div>
          <p v-if="!filteredSubscriptions.length" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">没有符合条件的订阅</p>
        </div>

        <div v-else-if="activeTab === 'nodes'" class="min-w-0 space-y-3">
          <div class="grid min-w-0 gap-2 md:grid-cols-[minmax(0,1fr)_minmax(0,12rem)_minmax(0,10rem)]">
            <div class="relative min-w-0">
              <Icon name="search" size="sm" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
              <input v-model.trim="nodeQuery" type="search" class="input min-w-0 pl-9" placeholder="搜索节点、地区或标签" aria-label="搜索 Mihomo 节点" />
            </div>
            <select v-model="nodeSubscriptionFilter" class="input min-w-0" aria-label="按订阅筛选节点">
              <option value="">全部订阅</option>
              <option v-for="subscription in subscriptions" :key="subscription.id" :value="String(subscription.id)">{{ subscription.name }}</option>
            </select>
            <select v-model="nodeHealthFilter" class="input min-w-0" aria-label="按状态筛选节点">
              <option value="">全部状态</option><option value="alive">可用</option><option value="down">异常</option><option value="unknown">未检测</option><option value="excluded">已排除</option>
            </select>
          </div>

          <div class="flex flex-wrap items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 p-2 dark:border-dark-700 dark:bg-dark-900/40">
            <span class="px-2 text-sm text-gray-600 dark:text-gray-300">当前结果 {{ selectableFilteredNodes.length }} 个 · 已选 {{ selectedNodeIDs.length }} 个</span>
            <button type="button" class="btn btn-secondary min-h-11 px-3" :disabled="!selectableFilteredNodes.length || allFilteredNodesSelected" @click="selectAllFilteredNodes">选择当前结果</button>
            <button type="button" class="btn btn-secondary min-h-11 px-3" :disabled="!selectedNodeIDs.length" @click="selectedNodeKeys = new Set()">清空选择</button>
            <button type="button" class="btn btn-primary min-h-11 px-3" :disabled="!selectedNodeIDs.length" @click="confirmNodeAction('create_dedicated_routes')">批量建专线</button>
            <button type="button" class="btn btn-secondary min-h-11 px-3" :disabled="!selectedNodeIDs.length" title="按所选节点所属订阅批量检测" @click="runNodeTest">批量检测所属订阅</button>
            <button type="button" class="btn btn-secondary min-h-11 px-3" :disabled="!selectedNodeIDs.length" @click="confirmNodeAction('enable')">批量启用</button>
            <button type="button" class="btn btn-secondary min-h-11 px-3" :disabled="!selectedNodeIDs.length" @click="confirmNodeAction('disable')">批量停用</button>
            <button type="button" class="btn btn-secondary min-h-11 px-3" :disabled="!selectedNodeIDs.length" @click="confirmNodeAction('exclude')">批量排除</button>
            <button type="button" class="btn btn-secondary min-h-11 px-3" :disabled="!selectedNodeIDs.length" @click="confirmNodeAction('restore')">批量恢复</button>
          </div>

          <div class="grid min-w-0 gap-3 md:grid-cols-2 2xl:grid-cols-3">
            <article v-for="node in filteredNodes" :key="String(nodeIdentity(node))" class="min-w-0 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
              <div class="flex min-w-0 items-start gap-3">
                <label class="flex h-11 w-11 shrink-0 cursor-pointer items-center justify-center" :title="`选择 ${node.display_name || node.name}`">
                  <input type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" :checked="isNodeSelected(node)" :disabled="Boolean(node.upstream_removed_at)" @change="toggleNode(node)" />
                </label>
                <div class="min-w-0 flex-1">
                  <div class="flex min-w-0 flex-wrap items-center gap-2">
                    <h3 class="truncate text-sm font-semibold text-gray-900 dark:text-white" :title="node.display_name || node.name">{{ node.display_name || node.name }}</h3>
                    <span :class="['badge', nodeStatusClass(node)]">{{ nodeStatusLabel(node) }}</span>
                  </div>
                  <p class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400" :title="node.subscription_name || '-'">{{ node.subscription_name || subscriptionName(node.subscription_id) }}</p>
                </div>
                <span class="shrink-0 text-xs tabular-nums text-gray-500">{{ delayLabel(node.delay) }}</span>
              </div>
              <div class="mt-3 flex min-w-0 flex-wrap gap-2 text-xs text-gray-500 dark:text-gray-400">
                <span v-if="node.region" class="badge badge-gray">{{ node.region }}</span>
                <span v-for="tag in node.tags || []" :key="tag" class="badge badge-gray">{{ tag }}</span>
                <span v-if="node.exit_ip" class="truncate font-mono" :title="node.exit_ip">{{ node.exit_ip }}</span>
              </div>
            </article>
          </div>
          <p v-if="!filteredNodes.length" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">没有符合条件的节点</p>
        </div>

        <div v-else-if="activeTab === 'routes'" class="min-w-0 space-y-3">
          <div class="grid min-w-0 gap-2 sm:grid-cols-2 xl:grid-cols-[minmax(0,1fr)_12rem_12rem_auto]">
            <div class="relative min-w-0">
              <Icon name="search" size="sm" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
              <input v-model.trim="routeQuery" type="search" class="input min-w-0 pl-9" placeholder="搜索线路、节点或出口 IP" aria-label="搜索 Mihomo 线路" />
            </div>
            <select v-model="routeKindFilter" class="input min-w-0" aria-label="按策略筛选线路"><option value="">全部策略</option><option v-for="option in routeKindOptions" :key="option.value" :value="option.value">{{ option.label }}</option></select>
            <select v-model="routeHealthFilter" class="input min-w-0" aria-label="按健康筛选线路"><option value="">全部健康状态</option><option value="healthy">健康</option><option value="degraded">降级</option><option value="failed">异常</option><option value="unknown">未检测</option></select>
            <button type="button" class="btn btn-primary min-h-11" @click="openRouteForm()"><Icon name="plus" size="sm" class="mr-1.5" />新建线路</button>
          </div>

          <div class="grid min-w-0 gap-3 2xl:grid-cols-2">
            <article
              v-for="route in filteredRoutes"
              :key="route.id"
              class="min-w-0 rounded-lg border bg-white p-3 transition-colors dark:bg-dark-800"
              :class="route.proxy_id === highlightedProxyID ? 'border-primary-500 ring-2 ring-primary-100 dark:ring-primary-900/30' : 'border-gray-200 dark:border-dark-700'"
            >
              <div class="flex min-w-0 items-start justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <div class="flex min-w-0 flex-wrap items-center gap-2">
                    <h3 class="truncate text-sm font-semibold text-gray-900 dark:text-white" :title="route.name">{{ route.name }}</h3>
                    <span class="badge badge-primary">{{ routeKindLabel(route.kind) }}</span>
                    <span :class="['badge', routeHealthClass(route.health)]">{{ routeHealthLabel(route.health) }}</span>
                  </div>
                  <p class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400" :title="routeSubscriptionNames(route)">{{ routeSubscriptionNames(route) }}</p>
                </div>
                <span :class="['badge shrink-0', route.enabled ? 'badge-success' : 'badge-gray']">{{ route.enabled ? '已启用' : '已停用' }}</span>
              </div>
              <dl class="mt-3 grid grid-cols-2 gap-3 text-xs sm:grid-cols-3">
                <div class="min-w-0"><dt class="text-gray-500">当前节点</dt><dd class="mt-1 truncate font-medium text-gray-800 dark:text-gray-100" :title="route.current_node || '-'">{{ route.current_node || '-' }}</dd></div>
                <div class="min-w-0"><dt class="text-gray-500">出口 IP</dt><dd class="mt-1 truncate font-mono font-medium text-gray-800 dark:text-gray-100" :title="route.exit_ip || '-'">{{ route.exit_ip || '-' }}</dd></div>
                <div><dt class="text-gray-500">延迟</dt><dd class="mt-1 font-medium text-gray-800 dark:text-gray-100">{{ delayLabel(route.latency_ms) }}</dd></div>
                <div><dt class="text-gray-500">托管代理</dt><dd class="mt-1 font-medium text-gray-800 dark:text-gray-100">#{{ route.proxy_id }}</dd></div>
                <div><dt class="text-gray-500">内部端口</dt><dd class="mt-1 font-medium text-gray-800 dark:text-gray-100">{{ route.listener_port }}</dd></div>
                <div><dt class="text-gray-500">绑定账号</dt><dd class="mt-1 font-medium text-gray-800 dark:text-gray-100">{{ route.account_count }}</dd></div>
              </dl>
              <div class="mt-3 grid grid-cols-2 gap-2 sm:flex sm:flex-wrap">
                <button type="button" class="btn btn-secondary min-h-11 px-3" @click="testOneRoute(route)"><Icon name="play" size="sm" class="mr-1.5" />测试</button>
                <button type="button" class="btn btn-secondary min-h-11 px-3" @click="openRouteForm(route)"><Icon name="edit" size="sm" class="mr-1.5" />编辑</button>
                <button type="button" class="btn btn-secondary min-h-11 px-3" @click="confirmRouteToggle(route)">{{ route.enabled ? '停用' : '启用' }}</button>
                <button v-if="route.account_count > 0" type="button" class="btn btn-secondary min-h-11 px-3" @click="emit('view-proxy-accounts', route.proxy_id)"><Icon name="users" size="sm" class="mr-1.5" />查看/迁移账号</button>
                <button type="button" class="btn btn-secondary min-h-11 px-3 text-red-600" :disabled="route.account_count > 0" :title="route.account_count > 0 ? '请先查看并迁移绑定账号' : '删除线路'" @click="confirmDeleteRoute(route)"><Icon name="trash" size="sm" class="mr-1.5" />删除</button>
              </div>
            </article>
          </div>
          <p v-if="!filteredRoutes.length" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">没有符合条件的线路</p>
        </div>

        <div v-else class="min-w-0 space-y-4">
          <dl class="grid min-w-0 gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700"><dt class="text-xs text-gray-500">Mihomo 版本</dt><dd class="mt-1 truncate font-medium text-gray-900 dark:text-white">{{ workbench.status.version || '-' }}</dd></div>
            <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700"><dt class="text-xs text-gray-500">控制器</dt><dd class="mt-1 font-medium text-gray-900 dark:text-white">{{ workbench.status.controller_connected === false ? '未连接' : '已连接' }}</dd></div>
            <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700"><dt class="text-xs text-gray-500">生成配置</dt><dd class="mt-1 font-medium text-gray-900 dark:text-white">{{ workbench.status.config_valid === false ? '需要检查' : '验证通过' }}</dd></div>
            <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700"><dt class="text-xs text-gray-500">最近重载</dt><dd class="mt-1 truncate font-medium text-gray-900 dark:text-white" :title="fullTime(workbench.status.last_reload_at)">{{ shortTime(workbench.status.last_reload_at) }}</dd></div>
          </dl>
          <div v-if="workbench.status.last_reload_error" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/20 dark:text-red-300">{{ workbench.status.last_reload_error }}</div>
          <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">出口概况</h3>
            <div class="mt-3 grid grid-cols-2 gap-3 text-sm xl:grid-cols-5">
              <div><span class="text-gray-500">健康线路</span><strong class="ml-2 text-gray-900 dark:text-white">{{ healthyRouteCount }}</strong></div>
              <div><span class="text-gray-500">异常线路</span><strong class="ml-2 text-gray-900 dark:text-white">{{ unhealthyRouteCount }}</strong></div>
              <div><span class="text-gray-500">未检测线路</span><strong class="ml-2 text-gray-900 dark:text-white">{{ unknownRouteCount }}</strong></div>
              <div><span class="text-gray-500">可用节点</span><strong class="ml-2 text-gray-900 dark:text-white">{{ aliveNodeCount }}</strong></div>
              <div><span class="text-gray-500">绑定账号</span><strong class="ml-2 text-gray-900 dark:text-white">{{ totalAccountCount }}</strong></div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </section>

  <BaseDialog :show="entityDialog !== null" :title="entityDialogTitle" width="normal" @close="closeEntityDialog">
    <form id="mihomo-entity-form" class="space-y-4" @submit.prevent="submitEntity">
      <template v-if="entityDialog === 'subscription'">
        <div><label for="mihomo-subscription-name" class="input-label">订阅名称</label><input id="mihomo-subscription-name" v-model.trim="subscriptionForm.name" required class="input" placeholder="例如：香港专线池" /></div>
        <div><label for="mihomo-subscription-url" class="input-label">订阅地址</label><input id="mihomo-subscription-url" v-model.trim="subscriptionForm.subscription_url" :required="!editingSubscription" type="url" autocomplete="off" class="input" :placeholder="editingSubscription ? '留空表示不修改' : 'https://…'" /><p class="input-hint mt-1">地址加密保存，页面只显示来源和掩码。</p></div>
        <div><label for="mihomo-refresh-interval" class="input-label">自动刷新周期（分钟）</label><input id="mihomo-refresh-interval" v-model.number="subscriptionForm.refresh_interval_minutes" required type="number" min="5" class="input" /></div>
        <label class="flex min-h-11 items-center gap-3"><input v-model="subscriptionForm.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" /><span class="text-sm text-gray-700 dark:text-gray-200">启用订阅</span></label>
      </template>
      <template v-else-if="entityDialog === 'route'">
        <div><label for="mihomo-route-name" class="input-label">线路名称</label><input id="mihomo-route-name" v-model.trim="routeForm.name" required class="input" placeholder="例如：香港定向 01" /></div>
        <div><label for="mihomo-route-kind" class="input-label">线路策略</label><select id="mihomo-route-kind" v-model="routeForm.kind" required class="input"><option v-for="option in routeKindOptions" :key="option.value" :value="option.value">{{ option.label }}</option></select><p class="input-hint mt-1">专线固定节点；最低延迟、容灾和动态线路可选择多个节点。</p></div>
        <fieldset><legend class="input-label">订阅范围</legend><div class="mt-1 max-h-36 overflow-y-auto rounded-lg border border-gray-200 p-2 dark:border-dark-700"><label v-for="subscription in subscriptions" :key="subscription.id" class="flex min-h-11 items-center gap-3 px-2"><input type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" :checked="routeForm.subscription_ids.includes(subscription.id)" @change="toggleRouteSubscription(subscription.id)" /><span class="min-w-0 truncate text-sm text-gray-700 dark:text-gray-200">{{ subscription.name }}</span></label></div></fieldset>
        <fieldset>
          <legend class="input-label">节点范围</legend>
          <div class="mt-1 grid gap-2 sm:grid-cols-2">
            <input v-model.trim="routeNodeQuery" type="search" class="input min-h-11 min-w-0" placeholder="搜索节点、地区或标签" aria-label="搜索线路节点" />
            <select v-model="routeNodeHealthFilter" class="input min-h-11 min-w-0" aria-label="按状态筛选线路节点"><option value="">全部状态</option><option value="alive">可用</option><option value="down">异常</option><option value="unknown">未检测</option></select>
          </div>
          <div class="mt-2 flex flex-wrap items-center gap-2">
            <span class="mr-auto text-sm text-gray-600 dark:text-gray-300">当前结果 {{ routeFormNodes.length }} 个 · 已选 {{ routeForm.node_ids.length }} 个</span>
            <button type="button" class="btn btn-secondary min-h-11 px-3" :disabled="!routeFormNodes.length" @click="toggleAllRouteFormNodes">{{ allRouteFormNodesSelected ? '取消当前结果' : '选择当前结果' }}</button>
            <button type="button" class="btn btn-secondary min-h-11 px-3" :disabled="!routeForm.node_ids.length" @click="routeForm.node_ids = []">清空</button>
          </div>
          <div class="mt-2 max-h-48 overflow-y-auto rounded-lg border border-gray-200 p-2 dark:border-dark-700">
            <label v-for="node in routeFormNodes" :key="String(nodeIdentity(node))" class="flex min-h-11 items-center gap-3 px-2">
              <input type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" :checked="routeNodeSelected(node)" @change="toggleRouteNode(node)" />
              <span class="min-w-0 flex-1 truncate text-sm text-gray-700 dark:text-gray-200">{{ node.display_name || node.name }}</span>
              <span :class="['badge shrink-0', nodeStatusClass(node)]">{{ nodeStatusLabel(node) }}</span>
              <span class="shrink-0 text-xs tabular-nums text-gray-500">{{ delayLabel(node.delay) }}</span>
            </label>
            <p v-if="!routeFormNodes.length" class="py-6 text-center text-sm text-gray-500 dark:text-gray-400">没有符合条件的节点</p>
          </div>
        </fieldset>
        <label class="flex min-h-11 items-center gap-3"><input v-model="routeForm.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" /><span class="text-sm text-gray-700 dark:text-gray-200">启用线路</span></label>
      </template>
      <div><label for="mihomo-entity-reason" class="input-label">{{ isPrimaryAdmin ? '变更原因（可选）' : '变更原因' }}</label><textarea id="mihomo-entity-reason" v-model.trim="entityReason" :required="!isPrimaryAdmin" rows="3" class="input" placeholder="说明用途、影响线路和账号范围"></textarea><p class="input-hint mt-1">{{ dialogPolicyHint }}</p></div>
    </form>
    <template #footer><div class="flex justify-end gap-2"><button type="button" class="btn btn-secondary min-h-11" @click="closeEntityDialog">取消</button><button type="submit" form="mihomo-entity-form" class="btn btn-primary min-h-11" :disabled="submitting || (!isPrimaryAdmin && !entityReason)">{{ submitting ? '处理中…' : changeActionLabel }}</button></div></template>
  </BaseDialog>

  <BaseDialog :show="confirmAction !== null" :title="confirmTitle" width="normal" @close="closeConfirmDialog">
    <form id="mihomo-confirm-form" class="space-y-4" @submit.prevent="submitConfirmedAction">
      <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm text-gray-700 dark:border-dark-700 dark:bg-dark-900/40 dark:text-gray-200">{{ confirmDescription }}</div>
      <div><label for="mihomo-confirm-reason" class="input-label">{{ isPrimaryAdmin ? '变更原因（可选）' : '变更原因' }}</label><textarea id="mihomo-confirm-reason" v-model.trim="confirmReason" :required="!isPrimaryAdmin" rows="3" class="input" placeholder="说明本次变更用途和影响范围"></textarea><p class="input-hint mt-1">{{ dialogPolicyHint }}</p></div>
    </form>
    <template #footer><div class="flex justify-end gap-2"><button type="button" class="btn btn-secondary min-h-11" @click="closeConfirmDialog">取消</button><button type="submit" form="mihomo-confirm-form" class="btn btn-primary min-h-11" :disabled="submitting || (!isPrimaryAdmin && !confirmReason)">{{ submitting ? '处理中…' : changeActionLabel }}</button></div></template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import { adminAPI } from '@/api/admin'
import type {
  MihomoApprovalResponse,
  MihomoLegacyImportPreview,
  MihomoNode,
  MihomoNodeActionInput,
  MihomoRoute,
  MihomoRouteInput,
  MihomoRouteKind,
  MihomoSubscription,
  MihomoSubscriptionInput,
  MihomoWorkbench
} from '@/api/admin/mihomo'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

type WorkbenchTab = 'subscriptions' | 'nodes' | 'routes' | 'runtime'
type ConfirmAction =
  | { type: 'legacy-import' }
  | { type: 'subscription-refresh'; subscription: MihomoSubscription }
  | { type: 'subscription-toggle'; subscription: MihomoSubscription }
  | { type: 'subscription-delete'; subscription: MihomoSubscription }
  | { type: 'route-toggle'; route: MihomoRoute }
  | { type: 'route-delete'; route: MihomoRoute }
  | { type: 'node'; action: MihomoNodeActionInput['action'] }

const emit = defineEmits<{
  'approval-submitted': []
  'routes-loaded': [routes: MihomoRoute[]]
  'view-proxy-accounts': [proxyID: number]
}>()
const appStore = useAppStore()
const authStore = useAuthStore()
const rootEl = ref<HTMLElement | null>(null)
const workbench = ref<MihomoWorkbench | null>(null)
const importPreview = ref<MihomoLegacyImportPreview | null>(null)
const loading = ref(false)
const submitting = ref(false)
const loadError = ref('')
const activeTab = ref<WorkbenchTab>('routes')
const highlightedProxyID = ref<number | null>(null)
const subscriptionQuery = ref('')
const nodeQuery = ref('')
const nodeSubscriptionFilter = ref('')
const nodeHealthFilter = ref('')
const routeQuery = ref('')
const routeKindFilter = ref('')
const routeHealthFilter = ref('')
const routeNodeQuery = ref('')
const routeNodeHealthFilter = ref('')
const selectedNodeKeys = ref<Set<string>>(new Set())
const entityDialog = ref<'subscription' | 'route' | null>(null)
const editingSubscription = ref<MihomoSubscription | null>(null)
const editingRoute = ref<MihomoRoute | null>(null)
const entityReason = ref('')
const confirmAction = ref<ConfirmAction | null>(null)
const confirmReason = ref('')

const subscriptionForm = reactive({ name: '', subscription_url: '', enabled: true, refresh_interval_minutes: 60 })
const routeForm = reactive<{ name: string; kind: MihomoRouteKind; subscription_ids: number[]; node_ids: Array<number | string>; enabled: boolean }>({ name: '', kind: 'dedicated', subscription_ids: [], node_ids: [], enabled: true })
const tabs: Array<{ value: WorkbenchTab; label: string }> = [
  { value: 'subscriptions', label: '订阅' }, { value: 'nodes', label: '节点' }, { value: 'routes', label: '线路' }, { value: 'runtime', label: '运行状态' }
]
const routeKindOptions: Array<{ value: MihomoRouteKind; label: string }> = [
  { value: 'dedicated', label: '专线' }, { value: 'automatic', label: '最低延迟' }, { value: 'fallback', label: '故障转移' }, { value: 'dynamic', label: '动态轮换' }, { value: 'directional', label: '定向' }
]

const subscriptions = computed(() => workbench.value?.subscriptions || [])
const nodes = computed(() => workbench.value?.nodes || [])
const routes = computed(() => workbench.value?.routes || [])
const runtimeHealthy = computed(() => Boolean(workbench.value?.status.enabled && workbench.value.status.configured && workbench.value.status.controller_connected !== false && workbench.value.status.config_valid !== false))
const isPrimaryAdmin = computed(() => authStore.user?.is_primary_admin === true)
const changeActionLabel = computed(() => isPrimaryAdmin.value ? '直接应用' : '提交审核')
const changePolicyHint = computed(() => isPrimaryAdmin.value ? '首位管理员的结构变更直接应用并记录审计。' : '结构变更由另一位管理员审核。')
const dialogPolicyHint = computed(() => isPrimaryAdmin.value ? '提交后直接应用并写入审计记录。' : '提交后由另一位管理员审核，审批详情会展示业务影响。')
const nodeHealth = (node: MihomoNode) => node.excluded ? 'excluded' : node.delay == null ? 'unknown' : node.alive ? 'alive' : 'down'
const healthyRouteCount = computed(() => routes.value.filter(route => route.health === 'healthy').length)
const unhealthyRouteCount = computed(() => routes.value.filter(route => route.health === 'failed' || route.health === 'degraded').length)
const unknownRouteCount = computed(() => routes.value.filter(route => route.health === 'unknown').length)
const aliveNodeCount = computed(() => nodes.value.filter(node => nodeHealth(node) === 'alive').length)
const totalAccountCount = computed(() => routes.value.reduce((total, route) => total + (route.account_count || 0), 0))
const filteredSubscriptions = computed(() => {
  const query = subscriptionQuery.value.toLocaleLowerCase()
  return subscriptions.value.filter(item => !query || [item.name, item.source_host, item.masked_url].some(value => value?.toLocaleLowerCase().includes(query)))
})
const filteredNodes = computed(() => {
  const query = nodeQuery.value.toLocaleLowerCase()
  return nodes.value.filter(node => {
    if (nodeSubscriptionFilter.value && String(node.subscription_id) !== nodeSubscriptionFilter.value) return false
    if (nodeHealthFilter.value && nodeHealth(node) !== nodeHealthFilter.value) return false
    return !query || [node.name, node.display_name, node.region, ...(node.tags || [])].some(value => value?.toLocaleLowerCase().includes(query))
  })
})
const filteredRoutes = computed(() => {
  const query = routeQuery.value.toLocaleLowerCase()
  return routes.value.filter(route => {
    if (routeKindFilter.value && route.kind !== routeKindFilter.value) return false
    if (routeHealthFilter.value && route.health !== routeHealthFilter.value) return false
    return !query || [route.name, route.current_node, route.exit_ip, routeSubscriptionNames(route)].some(value => value?.toLocaleLowerCase().includes(query))
  })
})
const selectedNodeIDs = computed(() => nodes.value.filter(node => !node.upstream_removed_at && isNodeSelected(node)).map(nodeIdentity))
const selectableFilteredNodes = computed(() => filteredNodes.value.filter(node => !node.upstream_removed_at))
const allFilteredNodesSelected = computed(() => selectableFilteredNodes.value.length > 0 && selectableFilteredNodes.value.every(isNodeSelected))
const routeFormNodes = computed(() => {
  const query = routeNodeQuery.value.toLocaleLowerCase()
  return nodes.value.filter(node => {
    if (node.upstream_removed_at || node.excluded || (routeForm.subscription_ids.length && !routeForm.subscription_ids.includes(node.subscription_id || 0))) return false
    if (routeNodeHealthFilter.value && nodeHealth(node) !== routeNodeHealthFilter.value) return false
    return !query || [node.name, node.display_name, node.region, node.exit_ip, node.subscription_name, ...(node.tags || [])].some(value => value?.toLocaleLowerCase().includes(query))
  })
})
const allRouteFormNodesSelected = computed(() => routeFormNodes.value.length > 0 && routeFormNodes.value.every(routeNodeSelected))
const entityDialogTitle = computed(() => entityDialog.value === 'subscription' ? `${editingSubscription.value ? '编辑' : '添加'}订阅` : `${editingRoute.value ? '编辑' : '新建'}线路`)
const confirmTitle = computed(() => {
  if (confirmAction.value?.type === 'legacy-import') return '确认导入现有配置'
  if (confirmAction.value?.type === 'subscription-refresh') return '确认刷新订阅'
  if (confirmAction.value?.type === 'node') return '批量节点操作'
  if (confirmAction.value?.type.includes('delete')) return '确认删除'
  return '确认状态变更'
})
const confirmDescription = computed(() => {
  const action = confirmAction.value
  if (!action) return ''
  if (action.type === 'legacy-import') return `导入 ${importPreview.value?.node_count || 0} 个节点和 ${importPreview.value?.route_count || 0} 条线路，保留现有端口、代理编号及账号绑定。`
  if (action.type === 'subscription-refresh') return `刷新订阅“${action.subscription.name}”，同步节点增删、可用状态、流量和到期信息。`
  if (action.type === 'subscription-toggle') return `${action.subscription.enabled ? '停用' : '启用'}订阅“${action.subscription.name}”，会影响其关联节点和线路。`
  if (action.type === 'subscription-delete') return `删除订阅“${action.subscription.name}”。存在关联线路时后端会阻止删除。`
  if (action.type === 'route-toggle') return `${action.route.enabled ? '停用' : '启用'}线路“${action.route.name}”，影响 ${action.route.account_count} 个账号。`
  if (action.type === 'route-delete') return `删除线路“${action.route.name}”。存在账号绑定时后端会阻止删除。`
  const labels: Record<MihomoNodeActionInput['action'], string> = { test: '测试', exclude: '排除', restore: '恢复', enable: '启用', disable: '停用', create_dedicated_routes: '分别创建专线' }
  return `对已选 ${selectedNodeIDs.value.length} 个节点执行“${labels[action.action]}”。`
})
const errorMessage = (error: any, fallback: string) => error?.message || error?.response?.data?.detail || fallback

const load = async () => {
  loading.value = true
  loadError.value = ''
  try {
    workbench.value = await adminAPI.mihomo.getWorkbench()
    emit('routes-loaded', workbench.value.routes || [])
    importPreview.value = null
    if (!workbench.value.subscriptions.length && !workbench.value.routes.length) {
      try { importPreview.value = await adminAPI.mihomo.getImportPreview() }
      catch (error: any) { loadError.value = errorMessage(error, '读取现有 Mihomo 配置失败') }
    }
  } catch (error: any) {
    loadError.value = errorMessage(error, '加载 Mihomo 工作台失败')
  } finally { loading.value = false }
}
const tabCount = (tab: WorkbenchTab) => tab === 'subscriptions' ? subscriptions.value.length : tab === 'nodes' ? nodes.value.length : tab === 'routes' ? routes.value.length : ''
const nodeIdentity = (node: MihomoNode): number | string => node.id ?? node.key ?? node.name
const nodeSelectionKey = (node: MihomoNode) => String(nodeIdentity(node))
const isNodeSelected = (node: MihomoNode) => selectedNodeKeys.value.has(nodeSelectionKey(node))
const toggleNode = (node: MihomoNode) => {
  if (node.upstream_removed_at) return
  const next = new Set(selectedNodeKeys.value)
  const key = nodeSelectionKey(node)
  next.has(key) ? next.delete(key) : next.add(key)
  selectedNodeKeys.value = next
}
const selectAllFilteredNodes = () => {
  const next = new Set(selectedNodeKeys.value)
  selectableFilteredNodes.value.forEach(node => next.add(nodeSelectionKey(node)))
  selectedNodeKeys.value = next
}
const subscriptionName = (id?: number) => subscriptions.value.find(item => item.id === id)?.name || '-'
const subscriptionBadge = (item: MihomoSubscription) => item.enabled && ['healthy', 'active'].includes(item.status) ? 'badge-success' : item.status === 'refreshing' ? 'badge-warning' : 'badge-danger'
const subscriptionStatusLabel = (item: MihomoSubscription) => !item.enabled ? '已停用' : item.status === 'refreshing' ? '刷新中' : ['healthy', 'active'].includes(item.status) ? '正常' : '异常'
const nodeStatusLabel = (node: MihomoNode) => node.upstream_removed_at ? '上游已移除' : ({ alive: '可用', down: '异常', unknown: '未检测', excluded: '已排除' })[nodeHealth(node)]
const nodeStatusClass = (node: MihomoNode) => node.upstream_removed_at ? 'badge-warning' : nodeHealth(node) === 'alive' ? 'badge-success' : nodeHealth(node) === 'unknown' ? 'badge-gray' : 'badge-danger'
const routeKindLabel = (kind: MihomoRouteKind) => routeKindOptions.find(item => item.value === kind)?.label || kind
const routeHealthLabel = (health: string) => ({ healthy: '健康', degraded: '降级', failed: '异常', unknown: '未检测' })[health] || health
const routeHealthClass = (health: string) => health === 'healthy' ? 'badge-success' : health === 'degraded' ? 'badge-warning' : health === 'failed' ? 'badge-danger' : 'badge-gray'
const routeSubscriptionNames = (route: MihomoRoute) => route.subscription_names?.join('、') || route.subscription_ids.map(subscriptionName).join('、') || '-'
const delayLabel = (delay?: number) => typeof delay === 'number' && delay >= 0 ? `${delay}ms` : '-'
const fullTime = (value?: string) => value ? new Date(value).toLocaleString() : '-'
const shortTime = (value?: string) => value ? new Date(value).toLocaleDateString() : '-'
const formatBytes = (value?: number) => {
  if (typeof value !== 'number') return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let current = value
  let unit = 0
  while (current >= 1024 && unit < units.length - 1) { current /= 1024; unit++ }
  return `${current >= 10 || unit === 0 ? current.toFixed(0) : current.toFixed(1)} ${units[unit]}`
}
const quotaLabel = (item: MihomoSubscription) => typeof item.total_bytes === 'number' ? `${formatBytes(item.used_bytes || 0)} / ${formatBytes(item.total_bytes)}` : '-'

const openSubscriptionForm = (subscription?: MihomoSubscription) => {
  editingSubscription.value = subscription || null
  Object.assign(subscriptionForm, { name: subscription?.name || '', subscription_url: '', enabled: subscription?.enabled ?? true, refresh_interval_minutes: subscription?.refresh_interval_minutes || 60 })
  entityReason.value = ''
  entityDialog.value = 'subscription'
}
const openRouteForm = (route?: MihomoRoute) => {
  editingRoute.value = route || null
  Object.assign(routeForm, { name: route?.name || '', kind: route?.kind || 'dedicated', subscription_ids: [...(route?.subscription_ids || [])], node_ids: [...(route?.node_ids || [])], enabled: route?.enabled ?? true })
  routeNodeQuery.value = ''
  routeNodeHealthFilter.value = ''
  entityReason.value = ''
  entityDialog.value = 'route'
}
const closeEntityDialog = () => { entityDialog.value = null; editingSubscription.value = null; editingRoute.value = null; entityReason.value = '' }
const toggleRouteSubscription = (id: number) => { routeForm.subscription_ids = routeForm.subscription_ids.includes(id) ? routeForm.subscription_ids.filter(item => item !== id) : [...routeForm.subscription_ids, id] }
const routeNodeSelected = (node: MihomoNode) => routeForm.node_ids.some(id => String(id) === String(nodeIdentity(node)))
const toggleRouteNode = (node: MihomoNode) => {
  const id = nodeIdentity(node)
  routeForm.node_ids = routeNodeSelected(node) ? routeForm.node_ids.filter(item => String(item) !== String(id)) : [...routeForm.node_ids, id]
}
const toggleAllRouteFormNodes = () => {
  const filteredKeys = new Set(routeFormNodes.value.map(nodeSelectionKey))
  if (allRouteFormNodesSelected.value) routeForm.node_ids = routeForm.node_ids.filter(id => !filteredKeys.has(String(id)))
  else {
    const selectedKeys = new Set(routeForm.node_ids.map(String))
    routeForm.node_ids = [...routeForm.node_ids, ...routeFormNodes.value.filter(node => !selectedKeys.has(nodeSelectionKey(node))).map(nodeIdentity)]
  }
}
const handleApprovalResult = async (result: MihomoApprovalResponse, directMessage: string) => {
  if (result.approval_required) { appStore.showSuccess('已提交给其他管理员审核'); emit('approval-submitted'); return }
  appStore.showSuccess(result.message || directMessage)
  await load()
}
const submitEntity = async () => {
  if (!entityDialog.value || (!isPrimaryAdmin.value && !entityReason.value)) return
  submitting.value = true
  try {
    let result: MihomoApprovalResponse
    if (entityDialog.value === 'subscription') {
      const input: MihomoSubscriptionInput = { ...subscriptionForm, reason: entityReason.value }
      result = editingSubscription.value ? await adminAPI.mihomo.updateWorkbenchSubscription(editingSubscription.value.id, input) : await adminAPI.mihomo.createSubscription(input)
    } else {
      const input: MihomoRouteInput = { ...routeForm, reason: entityReason.value }
      result = editingRoute.value ? await adminAPI.mihomo.updateRoute(editingRoute.value.id, input) : await adminAPI.mihomo.createRoute(input)
    }
    closeEntityDialog()
    await handleApprovalResult(result, 'Mihomo 变更已应用')
  } catch (error: any) { appStore.showError(errorMessage(error, '处理 Mihomo 变更失败')) } finally { submitting.value = false }
}
const testOneRoute = async (route: MihomoRoute) => {
  try { await adminAPI.mihomo.testRoute(route.id); appStore.showSuccess(`线路“${route.name}”测试完成`); await load() } catch (error: any) { appStore.showError(errorMessage(error, '线路测试失败')) }
}
const runNodeTest = async () => {
  try { await adminAPI.mihomo.runNodeAction({ action: 'test', node_ids: selectedNodeIDs.value }); appStore.showSuccess('节点测速完成'); await load() } catch (error: any) { appStore.showError(errorMessage(error, '节点测速失败')) }
}
const confirmSubscriptionRefresh = (subscription: MihomoSubscription) => { confirmAction.value = { type: 'subscription-refresh', subscription } }
const confirmSubscriptionToggle = (subscription: MihomoSubscription) => { confirmAction.value = { type: 'subscription-toggle', subscription } }
const confirmDeleteSubscription = (subscription: MihomoSubscription) => { confirmAction.value = { type: 'subscription-delete', subscription } }
const confirmRouteToggle = (route: MihomoRoute) => { confirmAction.value = { type: 'route-toggle', route } }
const confirmDeleteRoute = (route: MihomoRoute) => { confirmAction.value = { type: 'route-delete', route } }
const confirmNodeAction = (action: MihomoNodeActionInput['action']) => { confirmAction.value = { type: 'node', action } }
const closeConfirmDialog = () => { confirmAction.value = null; confirmReason.value = '' }
const submitConfirmedAction = async () => {
  const action = confirmAction.value
  if (!action || (!isPrimaryAdmin.value && !confirmReason.value)) return
  submitting.value = true
  try {
    let result: MihomoApprovalResponse
    if (action.type === 'legacy-import') result = await adminAPI.mihomo.importLegacy(confirmReason.value)
    else if (action.type === 'subscription-refresh') result = await adminAPI.mihomo.refreshSubscription(action.subscription.id, confirmReason.value)
    else if (action.type === 'subscription-toggle') {
      const item = action.subscription
      result = await adminAPI.mihomo.updateWorkbenchSubscription(item.id, { name: item.name, enabled: !item.enabled, refresh_interval_minutes: item.refresh_interval_minutes, reason: confirmReason.value })
    } else if (action.type === 'subscription-delete') result = await adminAPI.mihomo.deleteSubscription(action.subscription.id, confirmReason.value)
    else if (action.type === 'route-toggle') {
      const item = action.route
      result = await adminAPI.mihomo.updateRoute(item.id, { name: item.name, kind: item.kind, subscription_ids: item.subscription_ids, node_ids: item.node_ids, enabled: !item.enabled, reason: confirmReason.value })
    } else if (action.type === 'route-delete') result = await adminAPI.mihomo.deleteRoute(action.route.id, confirmReason.value)
    else result = await adminAPI.mihomo.runNodeAction({ action: action.action, node_ids: selectedNodeIDs.value, reason: confirmReason.value })
    closeConfirmDialog()
    await handleApprovalResult(result, 'Mihomo 变更已应用')
  } catch (error: any) { appStore.showError(errorMessage(error, '处理 Mihomo 变更失败')) } finally { submitting.value = false }
}
const openManagedProxy = async (proxyID: number) => {
  highlightedProxyID.value = proxyID
  activeTab.value = 'routes'
  routeQuery.value = ''
  routeKindFilter.value = ''
  routeHealthFilter.value = ''
  await nextTick()
  rootEl.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

defineExpose({ openManagedProxy, reload: load })
onMounted(load)
</script>
