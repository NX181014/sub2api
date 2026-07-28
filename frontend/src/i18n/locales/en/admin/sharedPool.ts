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
    approval: {
      title: 'Approval Center', subtitle: 'Account changes and credential access require another administrator to review',
      type: 'Request Type', requester: 'Requester', status: 'Status', requestedAt: 'Requested At', details: 'View changes',
      updateAccount: 'Account update', viewCredential: 'View credentials',
      pending: 'Pending', approved: 'Approved', rejected: 'Rejected', expired: 'Expired', consumed: 'Viewed', empty: 'No approval requests',
      field: 'Field', before: 'Before', after: 'After', decisionReason: 'Decision note', decisionReasonHint: 'Optional when approving; required when rejecting',
      approve: 'Approve', reject: 'Reject', submit: 'Submit for review', revealOnce: 'View once', verifyAndReveal: 'Verify and view',
      selfReviewBlocked: 'Requesters cannot review their own requests. Another administrator must decide.', rejectReasonRequired: 'Enter a reason before rejecting',
      approveSuccess: 'Request approved', rejectSuccess: 'Request rejected', credentialSubmitted: 'Credential access submitted for another administrator to review',
      credentialTitle: 'Account credentials · {name}', credentialHint: 'Enter the purpose. The requester, reviewer, purpose, and access time are audited.',
      purpose: 'Purpose', purposeHint: 'For example: diagnose upstream authentication failure', revealWarning: 'Credentials are shown once and cleared automatically after 60 seconds. Do not forward or save them to shared locations.',
      loadFailed: 'Failed to load approvals', decisionFailed: 'Failed to decide approval', submitFailed: 'Failed to submit approval', revealFailed: 'Failed to reveal credentials'
    },
    intake: {
      title: 'Shared-Pool Record', pending: 'Pending',
      preCreateTitle: 'Enter pool details before adding an account', preImportTitle: 'Enter pool details before importing accounts',
      prerequisiteHint: 'This draft is kept and saved to the pool after account creation. Failed writes can be retried.', importIdentityAuto: 'For batch imports, each new account name is recorded as its own upstream identity.',
      retryPending: '{count} account pool records failed to save. The draft is retained.', retryAction: 'Retry',
      retryFailed: '{count} account pool records failed to save. The draft is retained.', selectCreatedRetry: 'The new account could not be identified. Select it and retry saving.',
      pendingNotice: 'Account created. Complete its pool record before cost sharing and payback reporting.'
    },
    event: { title: 'Record Account Event', type: 'Event Type', date: 'Event Date', banned: 'Confirmed Ban', recovered: 'Recovered', refund: 'Refund', replaced: 'Replacement', retired: 'Retired', refundAmount: 'Refund Received', replacementAccount: 'Replacement Account', transferAmount: 'Transferred Cost', reason: 'Reason / Note', saved: 'Account event saved' },
    status: { active: 'Active', warning: 'Review', banned: 'Banned', inactive: 'Inactive', draft: 'Draft', locked: 'Locked', paid: 'Paid', recovered: 'Recovered', recovering: 'Recovering' },
    empty: { overview: 'No payback data for this period', settlement: 'No settlement data for this period', sources: 'No purchase source statistics' },
    errors: { load: 'Failed to load shared pool', options: 'Failed to load accounts or members', required: 'Complete the account identity, source, contributor, and uploader', replacementRequired: 'Select a replacement account', invalidCostPeriod: 'Cost must be positive and service end must not precede start', save: 'Failed to save account cost', event: 'Failed to save account event', fxRate: 'Value rate must be positive', lock: 'Failed to lock settlement' }
  }
}
