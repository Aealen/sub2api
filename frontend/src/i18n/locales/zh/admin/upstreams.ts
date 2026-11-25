export default {
  upstreams: {
    title: '上游管理',
    description: '维护上游供应商、上游侧分组与模型价格台账，横向比较各家进价',
    searchPlaceholder: '搜索上游名称',
    allStatus: '全部状态',
    create: '新建上游',
    compareEntry: '跨上游比价',
    empty: '还没有上游。先新建一个，或从「未归属账号」批量归并。',
    unassignedBanner: '有 {count} 个账号未归属上游',
    columns: {
      name: '上游',
      status: '状态',
      accounts: '账号数',
      balance: '整站余额',
      balanceUpdated: '最近同步',
      rateRange: '进价倍率',
      models: '台账模型',
      groups: '上游分组',
      actions: '操作'
    },
    status: {
      active: '启用',
      inactive: '停用'
    },
    actions: {
      detail: '详情',
      edit: '编辑',
      delete: '删除'
    },
    form: {
      createTitle: '新建上游',
      editTitle: '编辑上游',
      name: '名称',
      namePlaceholder: '例如 华龙 / PPDog / 勿忘',
      baseUrl: '上游地址',
      baseUrlPlaceholder: 'https://api.example.com',
      baseUrlHint: '只保留 scheme://host，用于把同源账号归并到一起',
      website: '后台 / 充值地址',
      contact: '联系人',
      note: '备注',
      status: '状态'
    },
    detail: {
      title: '上游详情',
      tabs: {
        accounts: '账号',
        groups: '上游分组',
        prices: '模型价格台账'
      },
      noAccounts: '该上游下还没有账号',
      noGroups: '还没有维护上游侧分组',
      noPrices: '还没有录入模型价格',
      probedRate: '探测倍率',
      probedHint: '来自账号的上游 /v1/sub2api/billing 探测',
      mappedModels: '{count} 个模型映射',
      contact: '联系人',
      website: '后台地址',
      note: '备注',
      baseUrl: '上游地址'
    },
    accounts: {
      name: '账号',
      platform: '平台',
      status: '状态',
      rate: '进价倍率',
      balance: '余额',
      groups: '平台分组',
      mapping: '模型映射'
    },
    groups: {
      add: '添加上游分组',
      edit: '编辑',
      name: '分组名',
      rate: '倍率',
      peak: '高峰倍率',
      peakWindow: '高峰时段',
      source: '来源',
      note: '备注',
      sourceValues: {
        manual: '手工',
        probe: '探测',
        import: '导入',
        crawler: '爬虫'
      }
    },
    prices: {
      add: '新增台账',
      edit: '编辑',
      import: '批量导入',
      upstreamModel: '上游模型名',
      modelName: '标准模型名',
      group: '上游分组',
      input: '输入价',
      output: '输出价',
      rate: '倍率',
      source: '来源',
      sourceValues: {
        manual: '手工',
        import: '导入',
        crawler: '爬虫'
      },
      perMillionHint: '单位：USD / 百万 token（保存时自动换算为 per-token）',
      importHint: '粘贴 JSON 数组，字段同单条台账；同一上游、同一分组、同一上游模型会被覆盖',
      importPlaceholder: '在此粘贴上游导出的 JSON 数组',
      unmatchedHint: '{count} 条未能对齐标准模型名，已按上游原名记录：{models}'
    },
    unassigned: {
      title: '未归属账号归并',
      hint: '按上游地址分组（同一 host 视为同一上游）；勾选后选择要挂到哪个上游',
      host: '上游地址',
      accounts: '账号数',
      sample: '包含',
      target: '挂到上游',
      createNew: '新建上游',
      assign: '确认归并',
      empty: '所有账号都已归属上游'
    },
    compare: {
      title: '跨上游比价',
      description: '行是模型，列是「供应商 × 分组」（同一供应商的不同分组各占一列）；格子是进价（USD / 百万 token）。台账价优先，缺失时按分组倍率或最低账号倍率推导',
      platform: '平台',
      allPlatforms: '全部平台',
      allModels: '全部模型',
      allUpstreams: '全部供应商',
      allGroups: '全部分组',
      noColumns: '没有匹配的列，请调整供应商或分组筛选',
      searchPlaceholder: '搜索模型',
      official: '目录价',
      cheapest: '最低价',
      empty: '没有可比较的数据：先给上游挂账号、维护分组倍率或录入台账',
      sourceLedger: '台账',
      sourceDerived: '推导',
      noData: '—',
      names: '上游',
      model: '模型'
    },
    overview: {
      title: '上游总览',
      description:
        '按「本站分组 / 上游」两个视角查看每家上游的整站余额、供货分组、售价与进价差，以及所选窗口内的真实毛利',
      searchPlaceholder: '搜索本站分组或上游',
      viewByGroup: '按本站分组',
      viewByUpstream: '按上游',
      windowDays: '{days} 天',
      balance: {
        title: '上游余额',
        siteBalance: '整站余额',
        sync: '最近同步',
        accounts: '账号数',
        groups: '上游分组',
        alert: '余额预警',
        empty: '还没有上游，先去「上游库」新建一个',
        expand: '展开全部 {count} 家上游',
        collapse: '收起'
      },
      columns: {
        group: '本站分组',
        platform: '平台',
        groupRate: '售价 D',
        upstream: '上游',
        status: '状态',
        balance: '整站余额',
        accounts: '账号数',
        upstreamGroup: '上游分组',
        purchaseRate: "进价 U'",
        derivedMargin: '推算毛利率',
        actualProfit: '实际毛利（该上游）',
        actualMargin: '实际毛利率',
        actions: '操作'
      },
      primary: '主力',
      noFlow: '无流水',
      noData: '—',
      link: {
        create: '建立关联',
        unlink: '解除关联',
        confirmTitle: '解除供货关联',
        confirmMessage:
          '确定解除本站分组「{group}」与上游「{upstream}」的分组「{upstreamGroup}」之间的关联？解除后该组合不再参与进价与毛利计算。',
        unlinked: '已解除供货关联',
        unlinkFailed: '解除供货关联失败'
      },
      dialog: {
        title: '建立供货关联',
        group: '本站分组',
        groupPlaceholder: '选择本站分组',
        upstreamGroup: '上游分组',
        upstreamGroupPlaceholder: '选择上游分组',
        upstreamGroupLoading: '正在加载上游分组…',
        upstreamGroupEmpty: '还没有可选的上游分组：先到「上游库」给上游添加分组，再回来关联',
        upstreamGroupFailed: '部分上游分组加载失败，请重试',
        note: '备注',
        notePlaceholder: '可选，例如：主力供货 / 备用',
        saved: '供货关联已保存',
        saveFailed: '保存供货关联失败'
      },
      rateDialog: {
        title: '编辑上游进价倍率',
        upstream: '上游',
        upstreamGroup: '上游分组',
        rate: '进价倍率',
        ratePlaceholder: '例如 0.3500',
        hint: '修改会同时影响比价页与推算毛利率',
        saved: '上游进价倍率已保存',
        saveFailed: '保存上游进价倍率失败',
        invalidRate: '请输入不小于 0 的有效数字'
      },
      unlinked: {
        title: '未建立上游关联的分组',
        hint: '这些本站分组还没有关联任何上游分组；点每一项右侧的「建立关联」即可就地完成，无需离开本页'
      },
      empty: '还没有可展示的供货关联：点上方「建立关联」把本站分组关联到上游分组',
      messages: {
        loadFailed: '加载上游总览失败'
      }
    },
    messages: {
      created: '上游已创建',
      saved: '已保存',
      deleted: '上游已删除',
      assigned: '已归并 {count} 个账号',
      imported: '已导入 {count} 条台账',
      loadFailed: '加载上游数据失败',
      saveFailed: '保存失败',
      deleteFailed: '删除失败',
      assignFailed: '归并失败',
      groupSaved: '上游分组已保存',
      priceSaved: '台账已保存',
      confirmDelete: '确定删除上游「{name}」？账号会保留，但会解除与该上游的归属关系。'
    }
  }
}
