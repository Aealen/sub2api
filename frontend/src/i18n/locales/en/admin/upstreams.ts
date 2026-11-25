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
      balance: 'Site balance',
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
    overview: {
      title: 'Upstream overview',
      description:
        'See each upstream as a site group view or an upstream view: site balance, supplying groups, the spread between sell and purchase rates, and the real margin in the selected window',
      searchPlaceholder: 'Search site group or upstream',
      viewByGroup: 'By site group',
      viewByUpstream: 'By upstream',
      windowDays: '{days} days',
      balance: {
        title: 'Upstream balances',
        siteBalance: 'Site balance',
        sync: 'Last sync',
        accounts: 'Accounts',
        groups: 'Upstream groups',
        alert: 'Low balance',
        empty: 'No upstreams yet. Create one in the upstream library first',
        expand: 'Show all {count} upstreams',
        collapse: 'Collapse'
      },
      columns: {
        group: 'Site group',
        platform: 'Platform',
        groupRate: 'Sell rate D',
        upstream: 'Upstream',
        status: 'Status',
        balance: 'Site balance',
        accounts: 'Accounts',
        upstreamGroup: 'Upstream group',
        purchaseRate: "Purchase rate U'",
        derivedMargin: 'Derived margin',
        actualProfit: 'Real profit (this upstream)',
        actualMargin: 'Real margin',
        actions: 'Actions'
      },
      primary: 'Primary',
      noFlow: 'No traffic yet',
      noData: '—',
      link: {
        create: 'Create link',
        unlink: 'Remove link',
        confirmTitle: 'Remove supply link',
        confirmMessage:
          'Remove the link between site group "{group}" and upstream "{upstream}" group "{upstreamGroup}"? Once removed, this pair no longer counts toward purchase rate and margin.',
        unlinked: 'Supply link removed',
        unlinkFailed: 'Failed to remove the supply link'
      },
      dialog: {
        title: 'Create supply link',
        group: 'Site group',
        groupPlaceholder: 'Select a site group',
        upstreamGroup: 'Upstream group',
        upstreamGroupPlaceholder: 'Select an upstream group',
        upstreamGroupLoading: 'Loading upstream groups…',
        upstreamGroupEmpty:
          'No upstream groups to pick yet. Add groups to an upstream in the upstream library, then come back to link them',
        upstreamGroupFailed: 'Some upstream groups failed to load. Try again',
        note: 'Note',
        notePlaceholder: 'Optional, e.g. primary supply / backup',
        saved: 'Supply link saved',
        saveFailed: 'Failed to save the supply link'
      },
      rateDialog: {
        title: 'Edit upstream purchase rate',
        upstream: 'Upstream',
        upstreamGroup: 'Upstream group',
        rate: 'Purchase rate',
        ratePlaceholder: 'e.g. 0.3500',
        hint: 'Changes affect both the price comparison page and the derived margin',
        saved: 'Upstream purchase rate saved',
        saveFailed: 'Failed to save the upstream purchase rate',
        invalidRate: 'Enter a valid number greater than or equal to 0'
      },
      unlinked: {
        title: 'Groups without an upstream link',
        hint: 'These site groups are not linked to any upstream group yet. Use "Create link" on each item to finish right here'
      },
      empty: 'No supply links to show yet. Use "Create link" above to link a site group to an upstream group',
      messages: {
        loadFailed: 'Failed to load the upstream overview'
      }
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
