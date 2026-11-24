export default {
  upstreams: {
    title: 'Upstreams',
    description: 'Maintain upstream vendors, their groups and model price ledgers, and compare purchase prices across vendors',
    searchPlaceholder: 'Search upstream name',
    allStatus: 'All statuses',
    create: 'New upstream',
    compareEntry: 'Price comparison',
    empty: 'No upstreams yet. Create one, or merge unassigned accounts in bulk.',
    unassignedBanner: '{count} accounts are not assigned to an upstream',
    columns: {
      name: 'Upstream',
      status: 'Status',
      accounts: 'Accounts',
      balance: 'Total balance',
      balanceUpdated: 'Last sync',
      rateRange: 'Purchase rate',
      models: 'Ledger models',
      groups: 'Upstream groups',
      actions: 'Actions'
    },
    status: {
      active: 'Active',
      inactive: 'Inactive'
    },
    actions: {
      detail: 'Details',
      edit: 'Edit',
      delete: 'Delete'
    },
    form: {
      createTitle: 'New upstream',
      editTitle: 'Edit upstream',
      name: 'Name',
      namePlaceholder: 'e.g. Hualong / PPDog / Wuwang',
      baseUrl: 'Upstream URL',
      baseUrlPlaceholder: 'https://api.example.com',
      baseUrlHint: 'Only scheme://host is kept; used to group accounts by origin',
      website: 'Console / top-up URL',
      contact: 'Contact',
      note: 'Note',
      status: 'Status'
    },
    detail: {
      title: 'Upstream details',
      tabs: {
        accounts: 'Accounts',
        groups: 'Upstream groups',
        prices: 'Model price ledger'
      },
      noAccounts: 'No accounts assigned to this upstream',
      noGroups: 'No upstream groups maintained yet',
      noPrices: 'No model prices recorded yet',
      probedRate: 'Probed rate',
      probedHint: 'From the upstream /v1/sub2api/billing probe on accounts',
      mappedModels: '{count} mapped models',
      contact: 'Contact',
      website: 'Console',
      note: 'Note',
      baseUrl: 'Upstream URL'
    },
    accounts: {
      name: 'Account',
      platform: 'Platform',
      status: 'Status',
      rate: 'Purchase rate',
      balance: 'Balance',
      groups: 'Platform groups',
      mapping: 'Model mapping'
    },
    groups: {
      add: 'Add upstream group',
      edit: 'Edit',
      name: 'Group name',
      rate: 'Rate',
      peak: 'Peak rate',
      peakWindow: 'Peak window',
      source: 'Source',
      note: 'Note',
      sourceValues: {
        manual: 'Manual',
        probe: 'Probe',
        import: 'Import',
        crawler: 'Crawler'
      }
    },
    prices: {
      add: 'Add ledger entry',
      edit: 'Edit',
      import: 'Bulk import',
      upstreamModel: 'Upstream model',
      modelName: 'Standard model',
      group: 'Upstream group',
      input: 'Input price',
      output: 'Output price',
      rate: 'Rate',
      source: 'Source',
      sourceValues: {
        manual: 'Manual',
        import: 'Import',
        crawler: 'Crawler'
      },
      perMillionHint: 'Unit: USD per 1M tokens (converted to per-token on save)',
      importHint: 'Paste a JSON array with the same fields as a single ledger entry; the same upstream/group/model will be overwritten',
      importPlaceholder: 'Paste the JSON array exported by the upstream',
      unmatchedHint: '{count} entries could not be matched to a standard model name and were kept under their upstream name: {models}'
    },
    unassigned: {
      title: 'Merge unassigned accounts',
      hint: 'Grouped by upstream URL (same host = same upstream). Select hosts, then choose the upstream to assign them to',
      host: 'Upstream URL',
      accounts: 'Accounts',
      sample: 'Includes',
      target: 'Assign to upstream',
      createNew: 'Create new upstream',
      assign: 'Confirm merge',
      empty: 'All accounts are already assigned to an upstream'
    },
    compare: {
      title: 'Cross-upstream price comparison',
      description: 'Rows are models, columns are vendor × group (each group of a vendor gets its own column); each cell is the purchase price (USD per 1M tokens). Ledger prices win; otherwise derived from the group rate or the lowest account rate',
      platform: 'Platform',
      allPlatforms: 'All platforms',
      allModels: 'All models',
      allUpstreams: 'All upstreams',
      allGroups: 'All groups',
      noColumns: 'No columns match the filters',
      searchPlaceholder: 'Search model',
      official: 'Catalog',
      cheapest: 'Cheapest',
      empty: 'Nothing to compare yet: assign accounts, maintain group rates or record ledger prices',
      sourceLedger: 'Ledger',
      sourceDerived: 'Derived',
      noData: '—',
      names: 'Upstreams',
      model: 'Model'
    },
    messages: {
      created: 'Upstream created',
      saved: 'Saved',
      deleted: 'Upstream deleted',
      assigned: 'Assigned {count} accounts',
      imported: 'Imported {count} ledger entries',
      loadFailed: 'Failed to load upstream data',
      saveFailed: 'Failed to save',
      deleteFailed: 'Failed to delete',
      assignFailed: 'Failed to assign accounts',
      groupSaved: 'Upstream group saved',
      priceSaved: 'Ledger entry saved',
      confirmDelete: 'Delete upstream "{name}"? Accounts are kept but detached from it.'
    }
  }
}
