import React, { useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Empty,
  Form,
  Input,
  Modal,
  Select,
  Tag,
  Table,
  Tabs,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import {
  copyInvoiceApplicationInfo,
  formatAmount,
  formatTime,
  InvoiceApplicationDetails,
  renderInvoiceProfileSummary,
  renderInvoiceStatusTag,
} from './invoiceShared';

const { TabPane } = Tabs;
const DEFAULT_PAGE_SIZE = 20;
const INVOICE_MIN_AMOUNT = 200;

const defaultProfileForm = {
  type: 'personal',
  title: '',
  tax_no: '',
  email: '',
  phone: '',
  address: '',
  bank_name: '',
  bank_account: '',
  is_default: false,
};

const hasRequiredInvoiceFields = (profile) =>
  !!profile?.title?.trim() &&
  !!profile?.tax_no?.trim() &&
  !!profile?.email?.trim();

const getInvoiceOrderDisabledReason = (order) =>
  order?.source_type === 'subscription' ? '订阅订单暂不支持开具发票' : '';

const getInvoiceOrderAmount = (order) => {
  const paidAmount = Number(order?.paid_amount ?? 0);
  if (paidAmount > 0) {
    return paidAmount;
  }
  const money = Number(order?.money ?? 0);
  if (money > 0) {
    return money;
  }
  return Number(order?.amount ?? 0);
};

const getInvoiceOrderCurrency = (order) => {
  const paidCurrency = String(order?.paid_currency || '').trim();
  if (paidCurrency) {
    return paidCurrency;
  }
  const paymentMethod = String(order?.payment_method || '')
    .trim()
    .toLowerCase();
  if (paymentMethod === 'stripe' || paymentMethod === 'creem') {
    return 'USD';
  }
  return 'CNY';
};

const InvoicePage = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [availableOrders, setAvailableOrders] = useState([]);
  const [applications, setApplications] = useState([]);
  const [records, setRecords] = useState([]);
  const [profiles, setProfiles] = useState([]);
  const [ordersPage, setOrdersPage] = useState(1);
  const [ordersPageSize, setOrdersPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [ordersTotal, setOrdersTotal] = useState(0);
  const [applicationsPage, setApplicationsPage] = useState(1);
  const [applicationsPageSize, setApplicationsPageSize] =
    useState(DEFAULT_PAGE_SIZE);
  const [applicationsTotal, setApplicationsTotal] = useState(0);
  const [recordsPage, setRecordsPage] = useState(1);
  const [recordsPageSize, setRecordsPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [recordsTotal, setRecordsTotal] = useState(0);
  const [selectedOrderIds, setSelectedOrderIds] = useState([]);
  const [selectedProfileId, setSelectedProfileId] = useState(undefined);
  const [applicationRemark, setApplicationRemark] = useState('');
  const [profileForm, setProfileForm] = useState(defaultProfileForm);
  const [detailApplication, setDetailApplication] = useState(null);

  const loadData = async ({
    nextOrdersPage = ordersPage,
    nextOrdersPageSize = ordersPageSize,
    nextApplicationsPage = applicationsPage,
    nextApplicationsPageSize = applicationsPageSize,
    nextRecordsPage = recordsPage,
    nextRecordsPageSize = recordsPageSize,
  } = {}) => {
    setLoading(true);
    try {
      const [ordersRes, applicationsRes, recordsRes, profilesRes] =
        await Promise.all([
          API.get('/api/invoice/order/self', {
            params: {
              p: nextOrdersPage,
              page_size: nextOrdersPageSize,
            },
          }),
          API.get('/api/invoice/application/self', {
            params: {
              p: nextApplicationsPage,
              page_size: nextApplicationsPageSize,
            },
          }),
          API.get('/api/invoice/record/self', {
            params: {
              p: nextRecordsPage,
              page_size: nextRecordsPageSize,
            },
          }),
          API.get('/api/invoice/profile'),
        ]);

      if (ordersRes.data.success) {
        const payload = ordersRes.data.data || {};
        setAvailableOrders(payload.items || []);
        setOrdersTotal(payload.total || 0);
        setOrdersPage(payload.page || nextOrdersPage);
        setOrdersPageSize(payload.page_size || nextOrdersPageSize);
      }
      if (applicationsRes.data.success) {
        const payload = applicationsRes.data.data || {};
        setApplications(payload.items || []);
        setApplicationsTotal(payload.total || 0);
        setApplicationsPage(payload.page || nextApplicationsPage);
        setApplicationsPageSize(payload.page_size || nextApplicationsPageSize);
      }
      if (recordsRes.data.success) {
        const payload = recordsRes.data.data || {};
        setRecords(payload.items || []);
        setRecordsTotal(payload.total || 0);
        setRecordsPage(payload.page || nextRecordsPage);
        setRecordsPageSize(payload.page_size || nextRecordsPageSize);
      }
      if (profilesRes.data.success) {
        const nextProfiles = profilesRes.data.data || [];
        setProfiles(nextProfiles);
        setSelectedProfileId((currentId) => {
          if (currentId && nextProfiles.some((item) => item.id === currentId)) {
            return currentId;
          }
          return (
            nextProfiles.find((item) => item.is_default)?.id ||
            nextProfiles[0]?.id
          );
        });
      }
    } catch (error) {
      showError(error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const orderColumns = useMemo(
    () => [
      {
        title: t('订单号'),
        dataIndex: 'trade_no',
      },
      {
        title: t('支付方式'),
        dataIndex: 'payment_method',
        render: (value, record) => (
          <div className='flex flex-wrap items-center gap-2'>
            <span>{value || '-'}</span>
            {record?.source_type === 'subscription' ? (
              <Tag color='orange'>{t('订阅')}</Tag>
            ) : null}
          </div>
        ),
      },
      {
        title: t('支付金额'),
        render: (_, record) =>
          formatAmount(
            record.paid_amount || record.money || record.amount,
            record.paid_currency,
          ),
      },
      {
        title: t('订单IP'),
        dataIndex: 'client_ip',
        render: (value) => value || '-',
      },
      {
        title: t('时间'),
        render: (_, record) =>
          formatTime(record.complete_time || record.create_time),
      },
    ],
    [t],
  );

  const applicationColumns = useMemo(
    () => [
      {
        title: t('申请单号'),
        dataIndex: 'application_no',
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        render: (value) => renderInvoiceStatusTag(value, t),
      },
      {
        title: t('开票信息'),
        render: (_, record) => renderInvoiceProfileSummary(record, t),
      },
      {
        title: t('金额'),
        render: (_, record) =>
          formatAmount(record.total_amount, record.currency),
      },
      {
        title: t('订单数'),
        render: (_, record) => record.items?.length || 0,
      },
      {
        title: t('驳回原因'),
        dataIndex: 'rejected_reason',
        render: (value) => value || '-',
      },
      {
        title: t('提交时间'),
        render: (_, record) => formatTime(record.created_at),
      },
      {
        title: t('操作'),
        render: (_, record) => (
          <div className='flex flex-wrap gap-2'>
            <Button
              size='small'
              theme='light'
              onClick={() => setDetailApplication(record)}
            >
              {t('查看信息')}
            </Button>
            <Button
              size='small'
              theme='borderless'
              onClick={() => copyInvoiceApplicationInfo(record, t)}
            >
              {t('复制信息')}
            </Button>
            <Button
              size='small'
              type='danger'
              theme='borderless'
              disabled={record.status !== 'pending_review'}
              onClick={async () => {
                try {
                  const res = await API.post(
                    `/api/invoice/application/${record.id}/cancel`,
                  );
                  if (res.data.success) {
                    showSuccess(t('申请已取消'));
                    loadData();
                  } else {
                    showError(res.data.message);
                  }
                } catch (error) {
                  showError(error);
                }
              }}
            >
              {t('取消')}
            </Button>
          </div>
        ),
      },
    ],
    [t],
  );

  const recordColumns = useMemo(
    () => [
      {
        title: t('发票号'),
        dataIndex: 'invoice_no',
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        render: (value) => renderInvoiceStatusTag(value, t),
      },
      {
        title: t('金额'),
        render: (_, record) =>
          formatAmount(record.total_amount, record.currency),
      },
      {
        title: t('申请数'),
        render: (_, record) => record.applications?.length || 0,
      },
      {
        title: t('开票时间'),
        render: (_, record) => formatTime(record.issued_at),
      },
    ],
    [t],
  );

  const profileColumns = useMemo(
    () => [
      {
        title: t('类型'),
        dataIndex: 'type',
      },
      {
        title: t('抬头'),
        dataIndex: 'title',
      },
      {
        title: t('税号'),
        dataIndex: 'tax_no',
        render: (value) => value || '-',
      },
      {
        title: t('邮箱'),
        dataIndex: 'email',
        render: (value) => value || '-',
      },
      {
        title: t('默认'),
        render: (_, record) =>
          record.is_default ? <Tag color='green'>{t('默认')}</Tag> : '-',
      },
      {
        title: t('操作'),
        render: (_, record) => (
          <Button
            size='small'
            type='danger'
            theme='borderless'
            onClick={async () => {
              try {
                const res = await API.delete(
                  `/api/invoice/profile/${record.id}`,
                );
                if (res.data.success) {
                  showSuccess(t('资料已删除'));
                  loadData();
                } else {
                  showError(res.data.message);
                }
              } catch (error) {
                showError(error);
              }
            }}
          >
            {t('删除')}
          </Button>
        ),
      },
    ],
    [t],
  );

  const rowSelection = {
    selectedRowKeys: selectedOrderIds,
    onChange: (keys) => setSelectedOrderIds(keys),
    getCheckboxProps: (record) => {
      const disabledReason = getInvoiceOrderDisabledReason(record);
      return {
        disabled: !!disabledReason,
      };
    },
    renderCell: ({ originNode, record, inHeader }) => {
      if (inHeader) {
        return originNode;
      }
      const disabledReason = getInvoiceOrderDisabledReason(record);
      if (!disabledReason) {
        return originNode;
      }
      return (
        <Tooltip content={t(disabledReason)} position='top' showArrow>
          <div className='inline-flex'>{originNode}</div>
        </Tooltip>
      );
    },
  };

  const profileOptions = profiles.map((item) => ({
    label: `${item.title}${item.is_default ? ` (${t('默认')})` : ''}`,
    value: item.id,
  }));

  const submitApplication = async () => {
    if (selectedOrderIds.length === 0) {
      showError(t('请先选择订单'));
      return;
    }
    const selectedOrders = availableOrders.filter((order) =>
      selectedOrderIds.includes(order.id),
    );
    if (selectedOrders.length === selectedOrderIds.length) {
      const disabledOrder = selectedOrders.find((order) =>
        getInvoiceOrderDisabledReason(order),
      );
      if (disabledOrder) {
        showError(t(getInvoiceOrderDisabledReason(disabledOrder)));
        return;
      }
      const currencies = new Set(
        selectedOrders.map((order) => getInvoiceOrderCurrency(order)),
      );
      if (currencies.size === 1) {
        const totalAmount = selectedOrders.reduce(
          (sum, order) => sum + getInvoiceOrderAmount(order),
          0,
        );
        if (totalAmount <= INVOICE_MIN_AMOUNT) {
          showError(t('开票申请总金额需大于 200 元，请在金额满足条件后再提交申请。'));
          return;
        }
      }
    }
    if (!selectedProfileId && !hasRequiredInvoiceFields(profileForm)) {
      showError(t('开票抬头、税号、邮箱为必填项'));
      return;
    }

    setSubmitting(true);
    try {
      const payload = {
        profile_id: selectedProfileId,
        topup_ids: selectedOrderIds,
        remark: applicationRemark,
        ...(!selectedProfileId ? profileForm : {}),
      };
      const res = await API.post('/api/invoice/application', payload);
      if (res.data.success) {
        showSuccess(t('开票申请已提交'));
        setSelectedOrderIds([]);
        setApplicationRemark('');
        loadData();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error);
    } finally {
      setSubmitting(false);
    }
  };

  const saveProfile = async () => {
    if (!hasRequiredInvoiceFields(profileForm)) {
      showError(t('开票抬头、税号、邮箱为必填项'));
      return;
    }

    setSubmitting(true);
    try {
      const res = await API.post('/api/invoice/profile', profileForm);
      if (res.data.success) {
        showSuccess(t('开票资料已保存'));
        setProfileForm(defaultProfileForm);
        loadData();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className='mt-[60px] px-2'>
      <Card bordered={false}>
        <Typography.Title heading={4}>{t('发票管理')}</Typography.Title>
        <Typography.Text type='secondary'>
          {t(
            '查看可开票订单、提交发票申请，并维护常用开票资料。抬头、税号、邮箱为必填项，审核通过后即视为开票成功。',
          )}
        </Typography.Text>
      </Card>

      <Tabs className='mt-4' type='line'>
        <TabPane tab={t('可开票订单')} itemKey='orders'>
          <Banner
            type='info'
            closeIcon={null}
            description={t(
              '仅展示支付成功且尚未进入开票流程的订单。订阅订单暂不支持开具发票，且开票申请总金额需大于 200 元。提交时必须填写抬头、税号、邮箱。',
            )}
          />
          <Card className='mt-4' loading={loading}>
            <div className='mb-4 flex flex-wrap gap-3'>
              <Select
                style={{ width: 320 }}
                placeholder={t('优先使用已保存资料')}
                optionList={profileOptions}
                value={selectedProfileId}
                onChange={(value) => setSelectedProfileId(value)}
                allowClear
              />
              <Input
                style={{ width: 280 }}
                placeholder={t('申请备注')}
                value={applicationRemark}
                onChange={setApplicationRemark}
              />
              <Button
                theme='solid'
                loading={submitting}
                onClick={submitApplication}
              >
                {t('提交开票申请')}
              </Button>
              <Button onClick={() => loadData()}>{t('刷新')}</Button>
            </div>

            {!selectedProfileId && (
              <Form layout='horizontal' className='mb-4'>
                <Form.Select
                  field='type'
                  label={t('类型')}
                  initValue={profileForm.type}
                  optionList={[
                    { label: t('个人'), value: 'personal' },
                    { label: t('企业'), value: 'company' },
                  ]}
                  onChange={(value) =>
                    setProfileForm((prev) => ({ ...prev, type: value }))
                  }
                />
                <Form.Input
                  field='title'
                  label={t('抬头')}
                  placeholder={t('必填')}
                  value={profileForm.title}
                  onChange={(value) =>
                    setProfileForm((prev) => ({ ...prev, title: value }))
                  }
                />
                <Form.Input
                  field='tax_no'
                  label={t('税号')}
                  placeholder={t('必填')}
                  value={profileForm.tax_no}
                  onChange={(value) =>
                    setProfileForm((prev) => ({ ...prev, tax_no: value }))
                  }
                />
                <Form.Input
                  field='email'
                  label={t('邮箱')}
                  placeholder={t('必填，用于接收发票')}
                  value={profileForm.email}
                  onChange={(value) =>
                    setProfileForm((prev) => ({ ...prev, email: value }))
                  }
                />
              </Form>
            )}

            <Table
              rowKey='id'
              columns={orderColumns}
              dataSource={availableOrders}
              rowSelection={rowSelection}
              pagination={{
                currentPage: ordersPage,
                pageSize: ordersPageSize,
                total: ordersTotal,
                showSizeChanger: true,
                pageSizeOpts: [10, 20, 50, 100],
                onPageChange: (page) => {
                  loadData({ nextOrdersPage: page });
                },
                onPageSizeChange: (pageSize) => {
                  loadData({
                    nextOrdersPage: 1,
                    nextOrdersPageSize: pageSize,
                  });
                },
              }}
              empty={
                <Empty
                  title={t('暂无可开票订单')}
                  description={t('支付成功后，订单会出现在这里。')}
                />
              }
            />
          </Card>
        </TabPane>

        <TabPane tab={t('开票申请')} itemKey='applications'>
          <Banner
            type='info'
            closeIcon={null}
            description={t(
              '每条申请都支持查看开票资料、订单明细，并可一键复制完整开票信息。审核通过后会直接进入开票成功。',
            )}
          />
          <Card className='mt-4' loading={loading}>
            <Table
              rowKey='id'
              columns={applicationColumns}
              dataSource={applications}
              pagination={{
                currentPage: applicationsPage,
                pageSize: applicationsPageSize,
                total: applicationsTotal,
                showSizeChanger: true,
                pageSizeOpts: [10, 20, 50, 100],
                onPageChange: (page) => {
                  loadData({ nextApplicationsPage: page });
                },
                onPageSizeChange: (pageSize) => {
                  loadData({
                    nextApplicationsPage: 1,
                    nextApplicationsPageSize: pageSize,
                  });
                },
              }}
              empty={<Empty title={t('暂无申请记录')} />}
            />
          </Card>

          <Card className='mt-4' title={t('已开票记录')} loading={loading}>
            <Table
              rowKey='id'
              columns={recordColumns}
              dataSource={records}
              pagination={{
                currentPage: recordsPage,
                pageSize: recordsPageSize,
                total: recordsTotal,
                showSizeChanger: true,
                pageSizeOpts: [10, 20, 50, 100],
                onPageChange: (page) => {
                  loadData({ nextRecordsPage: page });
                },
                onPageSizeChange: (pageSize) => {
                  loadData({
                    nextRecordsPage: 1,
                    nextRecordsPageSize: pageSize,
                  });
                },
              }}
              empty={<Empty title={t('暂无开票记录')} />}
            />
          </Card>
        </TabPane>

        <TabPane tab={t('开票资料')} itemKey='profiles'>
          <Card className='mt-4' title={t('新增资料')}>
            <Form layout='horizontal'>
              <Form.Select
                field='type'
                label={t('类型')}
                initValue={profileForm.type}
                optionList={[
                  { label: t('个人'), value: 'personal' },
                  { label: t('企业'), value: 'company' },
                ]}
                onChange={(value) =>
                  setProfileForm((prev) => ({ ...prev, type: value }))
                }
              />
              <Form.Input
                field='title'
                label={t('抬头')}
                placeholder={t('必填')}
                value={profileForm.title}
                onChange={(value) =>
                  setProfileForm((prev) => ({ ...prev, title: value }))
                }
              />
              <Form.Input
                field='tax_no'
                label={t('税号')}
                placeholder={t('必填')}
                value={profileForm.tax_no}
                onChange={(value) =>
                  setProfileForm((prev) => ({ ...prev, tax_no: value }))
                }
              />
              <Form.Input
                field='email'
                label={t('邮箱')}
                placeholder={t('必填，用于接收发票')}
                value={profileForm.email}
                onChange={(value) =>
                  setProfileForm((prev) => ({ ...prev, email: value }))
                }
              />
              <Form.Input
                field='phone'
                label={t('电话')}
                value={profileForm.phone}
                onChange={(value) =>
                  setProfileForm((prev) => ({ ...prev, phone: value }))
                }
              />
              <Form.Input
                field='address'
                label={t('地址')}
                value={profileForm.address}
                onChange={(value) =>
                  setProfileForm((prev) => ({ ...prev, address: value }))
                }
              />
              <Form.Input
                field='bank_name'
                label={t('开户行')}
                value={profileForm.bank_name}
                onChange={(value) =>
                  setProfileForm((prev) => ({ ...prev, bank_name: value }))
                }
              />
              <Form.Input
                field='bank_account'
                label={t('银行账号')}
                value={profileForm.bank_account}
                onChange={(value) =>
                  setProfileForm((prev) => ({
                    ...prev,
                    bank_account: value,
                  }))
                }
              />
              <Button theme='solid' loading={submitting} onClick={saveProfile}>
                {t('保存资料')}
              </Button>
            </Form>
          </Card>

          <Card className='mt-4' title={t('已保存资料')} loading={loading}>
            <Table
              rowKey='id'
              columns={profileColumns}
              dataSource={profiles}
              pagination={false}
              empty={<Empty title={t('暂无已保存资料')} />}
            />
          </Card>
        </TabPane>
      </Tabs>

      <Modal
        title={t('开票申请详情')}
        visible={!!detailApplication}
        onCancel={() => setDetailApplication(null)}
        footer={
          <div className='flex justify-end gap-2'>
            <Button
              theme='light'
              onClick={() =>
                detailApplication &&
                copyInvoiceApplicationInfo(detailApplication, t)
              }
            >
              {t('复制信息')}
            </Button>
            <Button onClick={() => setDetailApplication(null)}>
              {t('关闭')}
            </Button>
          </div>
        }
        width={860}
      >
        <InvoiceApplicationDetails application={detailApplication} t={t} />
      </Modal>
    </div>
  );
};

export default InvoicePage;
