export default {
  sharedPool: {
    title: '共享号池',
    description: '记录账号成本、按用量结算并跟踪回本情况',
    tabs: { overview: '回本概览', accounts: '账号列表', ledger: '成本台账', settlement: 'AA 结算', sources: '采购来源' },
    period: { label: '结算周期', day: '按天', week: '按周', month: '按月', custom: '自定义', timezone: 'Asia/Shanghai' },
    metrics: { purchaseCost: '采购成本', usageValue: '已回本成本', roiRate: '回本率', bannedLoss: '封禁损失', pendingRecovery: '待回本', activeAccounts: '活跃账号' },
    overview: { recoveryTitle: '号池回本进度', recoverySubtitle: '按每笔采购的预期 Token 和实际用量逐笔计算', accountsRecovered: '个账号已回本', accountRecovery: '账号回本明细', estimatedDays: '预计 {days} 天' },
    accounts: { title: '账号成本台账', subtitle: '记录贡献人、上传人、采购来源和服务期', addCost: '追加成本', recordEvent: '记录事件' },
    ledger: {
      writtenOffLoss: '核销损失',
      title: '账号成本台账', subtitle: '按账号查看成本摘要，按流水追溯每次采购、续费和调整',
      summaryView: '账号成本汇总', entriesView: '成本流水', batchAdd: '批量添加成本',
      searchPlaceholder: '搜索账号或上游身份...', searchEntries: '搜索账号、订单号或备注...', searchAccounts: '搜索要录入成本的账号...', selectedCount: '已选择 {count} 个账号',
      selectVisible: '选择当前结果', clearVisible: '取消当前结果', allUploaders: '全部上传人', allPayers: '全部付款人', allSources: '全部来源', allAccounts: '全部账号', allStatuses: '全部状态', allEntryTypes: '全部成本类型',
      costState: '成本状态', allCostStates: '全部成本状态', withCost: '已有成本', withoutCost: '未录入成本', startDate: '开始日期', endDate: '结束日期',
      payer: '付款人', netCost: '净成本', recognizedCost: '已摊销成本', costProgress: '成本消耗率', usageExpected: '用量 / 预期 Token', viewEntries: '查看流水',
      emptySummary: '暂无账号成本汇总', emptyEntries: '暂无成本流水', batchTitle: '批量添加账号成本',
      steps: { accounts: '选择账号', common: '公共资料', overrides: '逐账号覆盖', preview: '预览提交' },
      amountMode: '金额模式', perAccountAmount: '单个账号成本', perAccountHint: '填写的是每一个账号的成本', orderTotal: '订单总成本', orderTotalHint: '填写整张订单总额，再分配到账号',
      allocationMode: '订单分配方式', equalAllocation: '平均分配', manualAllocation: '手工分配', expectedTokens: '本次新增预期 Token', paidAt: '付款日期',
      perAccountPreview: '单价 {price} × {count} 个账号 = 本批总额 {total}', orderTotalPreview: '订单总额 {total}，分配到 {count} 个账号',
      overrideHint: '留空使用公共值；订单平均分配时，可固定部分账号金额，其余金额自动均分。', accountAmount: '该账号金额', accountCount: '账号数量',
      originalAmount: '原币金额', cnyAmount: '折合人民币', recordedAt: '入账时间', duplicateHint: '提交时后端会再次校验重复订单、重复账号和幂等键，整批成功后才入账。',
      previous: '上一步', next: '下一步', submitBatch: '确认批量入账', batchSaved: '已为 {count} 个账号入账，共 {total}',
      errors: {
        load: '加载成本台账失败', options: '加载筛选选项失败', accounts: '加载账号失败', submit: '批量入账失败',
        accounts_required: '请至少选择一个账号', duplicate_accounts: '选择结果中存在重复账号', payer_required: '请选择付款人', source_required: '请选择采购来源',
        amount_invalid: '金额必须大于 0', allocation_exceeds_total: '已固定的账号金额超过订单总额', allocation_total_mismatch: '手工分配金额合计必须等于订单总额',
        expected_tokens_invalid: '每个账号的预期 Token 必须是大于 0 的整数', period_invalid: '服务结束日期必须晚于开始日期'
      }
    },
    entryTypes: { purchase: '首次采购', renewal: '续费', topup: '充值', price_version: '价格变更', refund: '退款', adjustment: '成本调整', replacement_in: '补号转入', replacement_out: '补号转出', write_off: '核销损失' },
    settlement: { title: '本期 AA 结算', formula: '成员分摊 = 本期成本 x 个人用量占比 - 个人垫付', usageWeight: '计价用量', totalCost: '本期成本', carryForward: '结转', coverage: '计价覆盖率', unpricedWarning: '存在 {count} 条未计价用量，计价覆盖率达到 99% 后才可锁定。', payable: '应付', receivable: '应收', lock: '锁定结算', lockTitle: '锁定本期结算', lockMessage: '锁定后金额和计算快照保持不变，请确认本期数据已经核对。', lockedSuccess: '结算已锁定', pendingConfirmations: '还有 {count} 名成员待确认', pendingForYou: '待确认的共享号池账单', pendingForYouHint: '核对周期和金额后确认，全部成员确认后管理员才能结清。', confirmationNotRequired: '无需确认', pending: '待确认', confirmed: '已确认', confirmMine: '确认本人账单', resolveMember: '管理员代确认', confirmedSuccess: '账单已确认', markPaid: '确认结清', markPaidTitle: '确认本期已结清', markPaidMessage: '所有非零账单均已由成员本人或第一管理员确认，确定标记为已结清吗？', markedPaidSuccess: '本期结算已结清' },
    sources: { title: '采购来源质量', chartTitle: '来源回本率与 30 天封禁率', sampleHint: '按购买批次统计，可按上传人追溯到账号资料；样本较少时仅作参考', smallSample: '小样本', locateRecords: '定位号池资料' },
    columns: { account: '账号', accounts: '账号数', contributor: '贡献人', uploader: '上传人', source: '采购来源', costType: '成本类型', cost: '成本', servicePeriod: '服务期', warranty: '质保截止', status: '状态', roi: '回本率', remaining: '待回本', netProfit: '当前盈亏', recoveredAt: '首次回本时间', member: '成员', usageWeight: '计价用量', share: '用量占比', allocated: '分摊成本', credit: '垫付抵扣', net: '净额', confirmation: '成员确认', ban30: '30 天封禁率', survival: '平均存活天数', actions: '操作' },
    form: { title: '录入账号成本', account: '账号', providerIdentity: '上游账号身份', contributor: '贡献人/付款人', uploader: '实际上传人', source: '购买来源', purchaseUrl: '购买链接', orderNo: '订单号', entryType: '成本类型', cost: '实付成本', costPerAccount: '每个账号的实付成本', costPerAccountHint: '批量导入时，这个金额会分别记入每一个导入成功的账号。订单总价请先除以账号数。', currency: '币种', serviceStart: '服务开始', serviceEnd: '服务结束', warrantyEnd: '质保截止', notes: '备注', costSharingEnabled: '参与 AA 结算', costSharingEnabledHint: '关闭后该账号的成本和用量不进入后续结算周期', saved: '账号成本已保存', fxRate: 'USD/CNY 估值汇率', saveFxRate: '保存估值汇率', fxRateSaved: '估值汇率已保存' },
    actions: { poolRecord: '号池资料' },
    delete: {
      costDisposition: '剩余成本处理', costDispositionHint: '仅处理该账号尚未回本的成本；已摊销部分不会重复计损。',
		writeOff: '核销为损失', refund: '记录退款', transfer: '转移到补号', replacementAccount: '补号账号', refundAmount: '实际退款金额', refundAmountHint: '填写实际到账金额；未退款的待回本余额自动记为封禁损失。',
		reason: '删除原因', reasonHint: '必填，请记录封禁、过期或换号原因', auditHint: '账号运行数据会清理；已有成本、用量、审批和结算记录将保留用于审计。',
		loadAccountsFailed: '加载可选补号失败', bulkRequiresIndividual: '已选择 {count} 个账号。每个账号的剩余成本处理方式可能不同，请逐个删除并确认处置。', approvalSubmitted: '删除申请已提交，等待其他管理员审核', success: '账号已删除并完成关联数据处理', failed: '删除账号失败'
    },
    approval: {
      title: '审批中心', subtitle: '账号变更和凭证查看均由另一名管理员复核',
      type: '申请类型', requester: '申请人', status: '状态', requestedAt: '申请时间', details: '查看差异',
		updateAccount: '账号信息变更', viewCredential: '查看凭证', deleteAccount: '删除账号',
      pending: '待审核', approved: '已批准', rejected: '已驳回', expired: '已过期', consumed: '已查看', empty: '暂无审批申请',
      field: '字段', before: '变更前', after: '变更后', decisionReason: '审批意见', decisionReasonHint: '批准可选填，驳回时必须填写原因',
      approve: '批准', reject: '驳回', submit: '提交审核', revealOnce: '查看一次', verifyAndReveal: '验证后查看',
      selfReviewBlocked: '申请人不能审批自己的申请，请等待其他管理员处理。', rejectReasonRequired: '驳回时请填写原因',
      approveSuccess: '申请已批准', rejectSuccess: '申请已驳回', credentialSubmitted: '凭证查看申请已提交，等待其他管理员审核',
      credentialTitle: '账号凭证 · {name}', credentialHint: '填写本次查看用途；完整用途、申请人、审批人和查看时间都会留痕。', primaryDirectHint: '第一管理员可直接查看，系统仍会记录本次访问。',
      purpose: '查看用途', purposeHint: '例如：排查上游认证失败', revealWarning: '凭证只显示一次，并将在 60 秒后自动清除。请勿转发或保存到公共位置。',
      loadFailed: '加载审批申请失败', decisionFailed: '处理审批申请失败', submitFailed: '提交审批申请失败', revealFailed: '查看账号凭证失败'
    },
    intake: {
      title: '共享号池资料', pending: '待补录',
      preCreateTitle: '先录入号池资料，再添加账号', preImportTitle: '先录入号池资料，再导入账号',
      prerequisiteHint: '资料将作为草稿保留，账号创建成功后自动写入号池；写入失败可直接重试。', importIdentityAuto: '批量导入时，上游账号身份将按每个新账号的名称分别记录。',
      retryPending: '有 {count} 个账号的号池资料写入失败，草稿仍已保留。', retryAction: '重试写入',
      retryFailed: '{count} 个账号写入号池资料失败，草稿已保留', selectCreatedRetry: '未能定位新账号，请选择已创建的账号后重试保存',
      pendingNotice: '账号已创建，请在账号列表中补录号池资料后参与AA和回本统计'
    },
    event: { title: '记录账号事件', type: '事件类型', date: '发生日期', banned: '确认封禁', recovered: '恢复使用', refund: '退款', replaced: '换号', retired: '停用', refundAmount: '到账退款金额', replacementAccount: '替换账号', transferAmount: '转移成本', reason: '备注/原因', saved: '账号事件已保存' },
    status: { active: '活跃', warning: '关注', banned: '已封禁', inactive: '已停用', draft: '草稿', locked: '已锁定', paid: '已结清', recovered: '已回本', recovering: '回本中' },
    empty: { overview: '当前周期暂无回本数据', settlement: '当前周期暂无结算数据', sources: '暂无采购来源统计' },
    errors: { load: '加载共享号池失败', options: '加载账号或成员失败', required: '请完整填写账号身份、来源、贡献人和上传人', replacementRequired: '换号时请选择替换账号', invalidCostPeriod: '成本需大于 0，且服务结束日期不得早于开始日期', invalidExpectedTokens: '预期 Token 必须是大于 0 的整数', save: '保存账号成本失败', event: '保存账号事件失败', fxRate: '估值汇率需大于 0', lock: '锁定结算失败', confirmSettlement: '确认本人账单失败', markPaid: '结清本期账单失败' }
  }
}
