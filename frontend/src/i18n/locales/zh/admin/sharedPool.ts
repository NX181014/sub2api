export default {
  sharedPool: {
    title: '共享号池',
    description: '记录账号成本、按用量结算并跟踪回本情况',
    tabs: { overview: '回本概览', accounts: '账号列表', settlement: 'AA 结算', sources: '采购来源' },
    period: { label: '结算周期', day: '按天', week: '按周', month: '按月', custom: '自定义', timezone: 'Asia/Shanghai' },
    metrics: { purchaseCost: '采购成本', usageValue: '用量价值', roiRate: '回本率', bannedLoss: '封禁损失', pendingRecovery: '待回本', activeAccounts: '活跃账号' },
    overview: { recoveryTitle: '号池回本进度', recoverySubtitle: '按模型标准计价折算用量价值', accountsRecovered: '个账号已回本', accountRecovery: '账号回本明细', estimatedDays: '预计 {days} 天' },
    accounts: { title: '账号成本台账', subtitle: '记录贡献人、上传人、采购来源和服务期', addCost: '追加成本', recordEvent: '记录事件' },
    entryTypes: { purchase: '首次采购', renewal: '续费', topup: '充值', price_version: '价格变更', adjustment: '成本调整' },
    settlement: { title: '本期 AA 结算', formula: '成员分摊 = 本期成本 x 个人用量占比 - 个人垫付', usageWeight: '计价用量', totalCost: '本期成本', carryForward: '结转', coverage: '计价覆盖率', unpricedWarning: '存在 {count} 条未计价用量，计价覆盖率达到 99% 后才可锁定。', payable: '应付', receivable: '应收', lock: '锁定结算', lockTitle: '锁定本期结算', lockMessage: '锁定后金额和计算快照保持不变，请确认本期数据已经核对。', lockedSuccess: '结算已锁定' },
    sources: { title: '采购来源质量', chartTitle: '来源回本率与 30 天封禁率', sampleHint: '按购买批次统计，样本较少时仅作参考', smallSample: '小样本' },
    columns: { account: '账号', accounts: '账号数', contributor: '贡献人', uploader: '上传人', source: '采购来源', costType: '成本类型', cost: '成本', servicePeriod: '服务期', warranty: '质保截止', status: '状态', roi: '回本率', remaining: '待回本', netProfit: '当前盈亏', recoveredAt: '首次回本时间', member: '成员', usageWeight: '计价用量', share: '用量占比', allocated: '分摊成本', credit: '垫付抵扣', net: '净额', ban30: '30 天封禁率', survival: '平均存活天数', actions: '操作' },
    form: { title: '录入账号成本', account: '账号', providerIdentity: '上游账号身份', contributor: '贡献人/付款人', uploader: '实际上传人', source: '购买来源', purchaseUrl: '购买链接', orderNo: '订单号', entryType: '成本类型', cost: '实付成本', currency: '币种', serviceStart: '服务开始', serviceEnd: '服务结束', warrantyEnd: '质保截止', notes: '备注', saved: '账号成本已保存', fxRate: 'USD/CNY 估值汇率', saveFxRate: '保存估值汇率', fxRateSaved: '估值汇率已保存' },
    actions: { poolRecord: '号池资料' },
    approval: {
      title: '审批中心', subtitle: '账号变更和凭证查看均由另一名管理员复核',
      type: '申请类型', requester: '申请人', status: '状态', requestedAt: '申请时间', details: '查看差异',
      updateAccount: '账号信息变更', viewCredential: '查看凭证',
      pending: '待审核', approved: '已批准', rejected: '已驳回', expired: '已过期', consumed: '已查看', empty: '暂无审批申请',
      field: '字段', before: '变更前', after: '变更后', decisionReason: '审批意见', decisionReasonHint: '批准可选填，驳回时必须填写原因',
      approve: '批准', reject: '驳回', submit: '提交审核', revealOnce: '查看一次', verifyAndReveal: '验证后查看',
      selfReviewBlocked: '申请人不能审批自己的申请，请等待其他管理员处理。', rejectReasonRequired: '驳回时请填写原因',
      approveSuccess: '申请已批准', rejectSuccess: '申请已驳回', credentialSubmitted: '凭证查看申请已提交，等待其他管理员审核',
      credentialTitle: '账号凭证 · {name}', credentialHint: '填写本次查看用途；完整用途、申请人、审批人和查看时间都会留痕。',
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
    errors: { load: '加载共享号池失败', options: '加载账号或成员失败', required: '请完整填写账号身份、来源、贡献人和上传人', replacementRequired: '换号时请选择替换账号', invalidCostPeriod: '成本需大于 0，且服务结束日期不得早于开始日期', save: '保存账号成本失败', event: '保存账号事件失败', fxRate: '估值汇率需大于 0', lock: '锁定结算失败' }
  }
}
