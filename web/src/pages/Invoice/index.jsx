import React, { useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Empty,
  Form,
  Input,
  Select,
  Table,
  Tabs,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const { TabPane } = Tabs;
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

const formatAmount = (amount, currency) =>
  `${Number(amount || 0).toFixed(2)} ${currency || ''}`.trim();

const formatTime = (timestamp) => {
  if (!timestamp) return '-';
  return new Date(timestamp * 1000).toLocaleString();
};

const InvoicePage = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [availableOrders, setAvailableOrders] = useState([]);
  const [applications, setApplications] = useState([]);
  const [records, setRecords] = useState([]);
  const [profiles, setProfiles] = useState([]);
  const [selectedOrderIds, setSelectedOrderIds] = useState([]);
  const [selectedProfileId, setSelectedProfileId] = useState(undefined);
  const [applicationRemark, setApplicationRemark] = useState('');
  const [profileForm, setProfileForm] = useState(defaultProfileForm);

  const loadData = async () => {
    setLoading(true);
    try {
      const [ordersRes, applicationsRes, recordsRes, profilesRes] =
        await Promise.all([
          API.get('/api/invoice/order/self', {
            params: { p: 1, page_size: 100 },
          }),
          API.get('/api/invoice/application/self', {
            params: { p: 1, page_size: 100 },
          }),
          API.get('/api/invoice/record/self', {
            params: { p: 1, page_size: 100 },
          }),
          API.get('/api/invoice/profile'),
        ]);

      if (ordersRes.data.success) {
        setAvailableOrders(ordersRes.data.data.items || []);
      }
      if (applicationsRes.data.success) {
        setApplications(applicationsRes.data.data.items || []);
      }
      if (recordsRes.data.success) {
        setRecords(recordsRes.data.data.items || []);
      }
      if (profilesRes.data.success) {
        const nextProfiles = profilesRes.data.data || [];
        setProfiles(nextProfiles);
        const defaultProfile = nextProfiles.find((item) => item.is_default);
        if (defaultProfile) {
          setSelectedProfileId(defaultProfile.id);
        }
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
        render: (value) => value || '-',
      },
      {
        title: t('支付金额'),
        render: (_, record) =>
          formatAmount(record.paid_amount || record.money || record.amount, record.paid_currency),
      },
      {
        title: t('订单IP'),
        dataIndex: 'client_ip',
        render: (value) => value || '-',
      },
      {
        title: t('时间'),
        render: (_, record) => formatTime(record.complete_time || record.create_time),
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
        render: (value) => <Tag color='blue'>{value}</Tag>,
      },
      {
        title: t('金额'),
        render: (_, record) => formatAmount(record.total_amount, record.currency),
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
        title: t('时间'),
        render: (_, record) => formatTime(record.created_at),
      },
      {
        title: t('操作'),
        render: (_, record) => (
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
        render: (value) => <Tag color='green'>{value}</Tag>,
      },
      {
        title: t('金额'),
        render: (_, record) => formatAmount(record.total_amount, record.currency),
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
                const res = await API.delete(`/api/invoice/profile/${record.id}`);
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
          {t('查看可开票订单、提交发票申请，并维护常用开票资料。')}
        </Typography.Text>
      </Card>

      <Tabs className='mt-4' type='line'>
        <TabPane tab={t('可开票订单')} itemKey='orders'>
          <Banner
            type='info'
            closeIcon={null}
            description={t('仅展示支付成功且尚未进入开票流程的订单。')}
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
              <Button onClick={loadData}>{t('刷新')}</Button>
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
                  value={profileForm.title}
                  onChange={(value) =>
                    setProfileForm((prev) => ({ ...prev, title: value }))
                  }
                />
                <Form.Input
                  field='tax_no'
                  label={t('税号')}
                  value={profileForm.tax_no}
                  onChange={(value) =>
                    setProfileForm((prev) => ({ ...prev, tax_no: value }))
                  }
                />
                <Form.Input
                  field='email'
                  label={t('邮箱')}
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
              pagination={false}
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
          <Card className='mt-4' loading={loading}>
            <Table
              rowKey='id'
              columns={applicationColumns}
              dataSource={applications}
              pagination={false}
              empty={<Empty title={t('暂无申请记录')} />}
            />
          </Card>
          <Card className='mt-4' title={t('已开票记录')} loading={loading}>
            <Table
              rowKey='id'
              columns={recordColumns}
              dataSource={records}
              pagination={false}
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
                value={profileForm.title}
                onChange={(value) =>
                  setProfileForm((prev) => ({ ...prev, title: value }))
                }
              />
              <Form.Input
                field='tax_no'
                label={t('税号')}
                value={profileForm.tax_no}
                onChange={(value) =>
                  setProfileForm((prev) => ({ ...prev, tax_no: value }))
                }
              />
              <Form.Input
                field='email'
                label={t('邮箱')}
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
                  setProfileForm((prev) => ({ ...prev, bank_account: value }))
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
    </div>
  );
};

export default InvoicePage;
