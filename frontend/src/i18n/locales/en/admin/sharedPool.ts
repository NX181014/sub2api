export default {
  sharedPool: {
    title: 'Shared Account Pool',
    description: 'Track account costs, allocate usage, and monitor payback',
    tabs: { overview: 'Payback', accounts: 'Accounts', settlement: 'Cost Sharing', sources: 'Sources' },
    period: { label: 'Period', day: 'Daily', week: 'Weekly', month: 'Monthly', custom: 'Custom', timezone: 'Asia/Shanghai' },
    metrics: { purchaseCost: 'Purchase Cost', usageValue: 'Usage Value', roiRate: 'Payback Rate', bannedLoss: 'Ban Loss', pendingRecovery: 'Unrecovered', activeAccounts: 'Active Accounts' },
    overview: { recoveryTitle: 'Pool Payback', recoverySubtitle: 'Usage value based on standard model pricing', accountsRecovered: 'accounts recovered', accountRecovery: 'Account Payback Details', estimatedDays: 'Est. {days} days' },
    accounts: { title: 'Account Cost Ledger', subtitle: 'Track contributor, uploader, source, and service period', addCost: 'Add Account Cost', recordEvent: 'Record Event' },
    entryTypes: { purchase: 'Purchase', renewal: 'Renewal', topup: 'Top-up', price_version: 'Price Change', adjustment: 'Adjustment' },
    settlement: { title: 'Period Cost Sharing', formula: 'Member share = period cost x usage ratio - contributed cost', usageWeight: 'Priced Usage', totalCost: 'Period Cost', carryForward: 'Carry Forward', coverage: 'Pricing Coverage', unpricedWarning: '{count} usage records are unpriced. Coverage must reach 99% before locking.', payable: 'Payable', receivable: 'Receivable', lock: 'Lock Settlement', lockTitle: 'Lock Settlement', lockMessage: 'Amounts and calculation snapshots stay fixed after locking. Confirm the period data first.', lockedSuccess: 'Settlement locked' },
    sources: { title: 'Purchase Source Quality', chartTitle: 'Payback and 30-Day Ban Rate', sampleHint: 'Grouped by purchase source; small samples are directional only', smallSample: 'Small sample' },
    columns: { account: 'Account', accounts: 'Accounts', contributor: 'Contributor', uploader: 'Uploader', source: 'Source', costType: 'Cost Type', cost: 'Cost', servicePeriod: 'Service Period', warranty: 'Warranty End', status: 'Status', roi: 'Payback', remaining: 'Unrecovered', netProfit: 'Current P/L', recoveredAt: 'First Payback At', member: 'Member', usageWeight: 'Priced Usage', share: 'Usage Share', allocated: 'Allocated Cost', credit: 'Contribution Credit', net: 'Net Amount', ban30: '30-Day Ban', survival: 'Avg. Survival Days', actions: 'Actions' },
    form: { title: 'Add Account Cost', account: 'Account', providerIdentity: 'Provider Identity', contributor: 'Contributor / Payer', uploader: 'Uploader', source: 'Purchase Source', purchaseUrl: 'Purchase URL', orderNo: 'Order Number', entryType: 'Cost Type', cost: 'Paid Cost', currency: 'Currency', serviceStart: 'Service Start', serviceEnd: 'Service End', warrantyEnd: 'Warranty End', notes: 'Notes', saved: 'Account cost saved', fxRate: 'USD/CNY Value Rate', saveFxRate: 'Save Value Rate', fxRateSaved: 'Value rate saved' },
    actions: { poolRecord: 'Pool Record' },
    intake: { title: 'Complete Shared-Pool Record', pendingNotice: 'Account created. Complete its pool record before cost sharing and payback reporting.' },
    event: { title: 'Record Account Event', type: 'Event Type', date: 'Event Date', banned: 'Confirmed Ban', recovered: 'Recovered', refund: 'Refund', replaced: 'Replacement', retired: 'Retired', refundAmount: 'Refund Received', replacementAccount: 'Replacement Account', transferAmount: 'Transferred Cost', reason: 'Reason / Note', saved: 'Account event saved' },
    status: { active: 'Active', warning: 'Review', banned: 'Banned', inactive: 'Inactive', draft: 'Draft', locked: 'Locked', paid: 'Paid', recovered: 'Recovered', recovering: 'Recovering' },
    empty: { overview: 'No payback data for this period', settlement: 'No settlement data for this period', sources: 'No purchase source statistics' },
    errors: { load: 'Failed to load shared pool', options: 'Failed to load accounts or members', required: 'Select an account, contributor, and uploader', replacementRequired: 'Select a replacement account', invalidCostPeriod: 'Cost must be positive and service end must not precede start', save: 'Failed to save account cost', event: 'Failed to save account event', fxRate: 'Value rate must be positive', lock: 'Failed to lock settlement' }
  }
}
