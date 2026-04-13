import React from 'react';
import { Button, Card, Descriptions, Tag } from '@douyinfe/semi-ui';
import { copy, showError, showSuccess } from '../../helpers';

export const formatAmount = (amount, currency) =>
  `${Number(amount || 0).toFixed(2)} ${currency || ''}`.trim();

export const formatTime = (timestamp) => {
  if (!timestamp) return '-';
  return new Date(timestamp * 1000).toLocaleString();
};

const applicationStatusConfig = {
  pending_review: { color: 'blue', label: '待审核' },
  approved: { color: 'green', label: '已通过' },
  rejected: { color: 'red', label: '已驳回' },
  issued: { color: 'violet', label: '已开票' },
  cancelled: { color: 'grey', label: '已取消' },
  voided: { color: 'grey', label: '已作废' },
};

export const getInvoiceTypeLabel = (type, t) => {
  if (type === 'company') {
    return t('企业');
  }
  return t('个人');
};

export const renderInvoiceStatusTag = (status, t) => {
  const config = applicationStatusConfig[status] || {
    color: 'blue',
    label: status || '-',
  };
  return <Tag color={config.color}>{t(config.label)}</Tag>;
};

export const parseInvoiceProfileSnapshot = (application) => {
  if (!application?.profile_snapshot) {
    return {};
  }
  try {
    return JSON.parse(application.profile_snapshot);
  } catch (error) {
    return {};
  }
};

const buildInvoiceApplicationOrdersText = (application) => {
  const items = application?.items || [];
  if (items.length === 0) {
    return '-';
  }
  return items
    .map(
      (item, index) =>
        `${index + 1}. ${item.order_trade_no || '-'} | ${formatAmount(item.amount, item.currency)}`,
    )
    .join('\n');
};

export const buildInvoiceApplicationCopyText = (application, t) => {
  if (!application) {
    return '';
  }
  const snapshot = parseInvoiceProfileSnapshot(application);
  const lines = [
    `${t('申请单号')}: ${application.application_no || '-'}`,
    application.user_id ? `${t('用户ID')}: ${application.user_id}` : '',
    `${t('状态')}: ${t(applicationStatusConfig[application.status]?.label || application.status || '-')}`,
    `${t('申请金额')}: ${formatAmount(application.total_amount, application.currency)}`,
    `${t('提交时间')}: ${formatTime(application.created_at)}`,
    `${t('开票类型')}: ${getInvoiceTypeLabel(snapshot.type, t)}`,
    `${t('开票抬头')}: ${snapshot.title || '-'}`,
    `${t('税号')}: ${snapshot.tax_no || '-'}`,
    `${t('邮箱')}: ${snapshot.email || '-'}`,
    `${t('电话')}: ${snapshot.phone || '-'}`,
    `${t('地址')}: ${snapshot.address || '-'}`,
    `${t('开户行')}: ${snapshot.bank_name || '-'}`,
    `${t('银行账号')}: ${snapshot.bank_account || '-'}`,
    `${t('申请备注')}: ${application.remark || '-'}`,
    `${t('审核备注')}: ${application.admin_remark || '-'}`,
    `${t('驳回原因')}: ${application.rejected_reason || '-'}`,
    `${t('订单明细')}:\n${buildInvoiceApplicationOrdersText(application)}`,
  ];
  return lines.filter(Boolean).join('\n');
};

export const copyInvoiceApplicationInfo = async (application, t) => {
  const content = buildInvoiceApplicationCopyText(application, t);
  if (!content) {
    showError(t('暂无可复制的开票信息'));
    return;
  }
  const ok = await copy(content);
  if (ok) {
    showSuccess(t('开票信息已复制'));
  } else {
    showError(t('复制失败，请手动复制'));
  }
};

export const copyInvoiceField = async (value, successMessage, t) => {
  if (!value) {
    showError(t('暂无可复制内容'));
    return;
  }
  const ok = await copy(value);
  if (ok) {
    showSuccess(successMessage);
  } else {
    showError(t('复制失败，请手动复制'));
  }
};

export const renderInvoiceProfileSummary = (application, t) => {
  const snapshot = parseInvoiceProfileSnapshot(application);
  return (
    <div className='min-w-[180px]'>
      <div className='font-medium break-all'>{snapshot.title || '-'}</div>
      <div className='mt-1 flex flex-wrap items-center gap-2 text-xs text-[var(--semi-color-text-2)]'>
        <Tag
          color={snapshot.type === 'company' ? 'green' : 'blue'}
          size='small'
        >
          {getInvoiceTypeLabel(snapshot.type, t)}
        </Tag>
        {snapshot.tax_no ? (
          <span className='break-all'>{snapshot.tax_no}</span>
        ) : (
          <span>{t('无税号')}</span>
        )}
      </div>
    </div>
  );
};

export const InvoiceApplicationDetails = ({
  application,
  t,
  showUserId = false,
}) => {
  const snapshot = parseInvoiceProfileSnapshot(application);
  const items = application?.items || [];

  if (!application) {
    return null;
  }

  const summaryItems = [
    {
      key: t('申请单号'),
      value: application.application_no || '-',
    },
    showUserId
      ? {
          key: t('用户ID'),
          value: application.user_id || '-',
        }
      : null,
    {
      key: t('状态'),
      value: renderInvoiceStatusTag(application.status, t),
    },
    {
      key: t('申请金额'),
      value: formatAmount(application.total_amount, application.currency),
    },
    {
      key: t('订单数'),
      value: items.length,
    },
    {
      key: t('提交时间'),
      value: formatTime(application.created_at),
    },
    {
      key: t('审核时间'),
      value: formatTime(application.reviewed_at),
    },
  ].filter(Boolean);

  const invoiceInfoItems = [
    {
      key: t('开票类型'),
      value: getInvoiceTypeLabel(snapshot.type, t),
    },
    {
      key: t('开票抬头'),
      value: snapshot.title || '-',
    },
    {
      key: t('税号'),
      value: snapshot.tax_no || '-',
    },
    {
      key: t('邮箱'),
      value: snapshot.email || '-',
    },
    {
      key: t('电话'),
      value: snapshot.phone || '-',
    },
    {
      key: t('地址'),
      value: snapshot.address || '-',
    },
    {
      key: t('开户行'),
      value: snapshot.bank_name || '-',
    },
    {
      key: t('银行账号'),
      value: snapshot.bank_account || '-',
    },
  ];

  const remarkItems = [
    {
      key: t('申请备注'),
      value: application.remark || '-',
    },
    {
      key: t('审核备注'),
      value: application.admin_remark || '-',
    },
    {
      key: t('驳回原因'),
      value: application.rejected_reason || '-',
    },
  ];

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap gap-2'>
        <Button
          size='small'
          theme='solid'
          onClick={() => copyInvoiceApplicationInfo(application, t)}
        >
          {t('复制完整信息')}
        </Button>
        <Button
          size='small'
          onClick={() =>
            copyInvoiceField(
              formatAmount(application.total_amount, application.currency),
              t('订单金额已复制'),
              t,
            )
          }
        >
          {t('复制订单金额')}
        </Button>
        <Button
          size='small'
          onClick={() =>
            copyInvoiceField(snapshot.title, t('开票抬头已复制'), t)
          }
        >
          {t('复制抬头')}
        </Button>
        <Button
          size='small'
          onClick={() => copyInvoiceField(snapshot.tax_no, t('税号已复制'), t)}
        >
          {t('复制税号')}
        </Button>
        <Button
          size='small'
          onClick={() => copyInvoiceField(snapshot.email, t('邮箱已复制'), t)}
        >
          {t('复制邮箱')}
        </Button>
        <Button
          size='small'
          onClick={() =>
            copyInvoiceField(snapshot.bank_account, t('银行账号已复制'), t)
          }
        >
          {t('复制银行账号')}
        </Button>
      </div>

      <Card title={t('申请摘要')} bodyStyle={{ padding: 16 }}>
        <Descriptions data={summaryItems} />
      </Card>

      <Card title={t('开票资料')} bodyStyle={{ padding: 16 }}>
        <Descriptions data={invoiceInfoItems} />
      </Card>

      <Card title={t('备注信息')} bodyStyle={{ padding: 16 }}>
        <Descriptions data={remarkItems} />
      </Card>

      <Card title={t('订单明细')} bodyStyle={{ padding: 16 }}>
        {items.length === 0 ? (
          <div className='text-[var(--semi-color-text-2)]'>
            {t('暂无订单明细')}
          </div>
        ) : (
          <div className='space-y-2'>
            {items.map((item) => (
              <div
                key={item.id}
                className='flex flex-wrap items-center justify-between gap-3 rounded-lg border border-[var(--semi-color-border)] px-3 py-2'
              >
                <div className='break-all'>
                  <div className='font-medium'>
                    {item.order_trade_no || '-'}
                  </div>
                  <div className='text-xs text-[var(--semi-color-text-2)]'>
                    TopUp ID: {item.topup_id || '-'}
                  </div>
                </div>
                <Tag color='blue'>
                  {formatAmount(item.amount, item.currency)}
                </Tag>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
};
