import React, { useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Empty,
  Input,
  Modal,
  Table,
  Tabs,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const { TabPane } = Tabs;

const formatAmount = (amount, currency) =>
  `${Number(amount || 0).toFixed(2)} ${currency || ''}`.trim();

const formatTime = (timestamp) => {
  if (!timestamp) return '-';
  return new Date(timestamp * 1000).toLocaleString();
};

const InvoiceAdminPage = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [issuing, setIssuing] = useState(false);
  const [applications, setApplications] = useState([]);
  const [records, setRecords] = useState([]);
  const [selectedApplicationIds, setSelectedApplicationIds] = useState([]);
  const [invoiceNo, setInvoiceNo] = useState('');
  const [fileURL, setFileURL] = useState('');
  const [remark, setRemark] = useState('');

  const loadData = async () => {
    setLoading(true);
    try {
      const [applicationsRes, recordsRes] = await Promise.all([
        API.get('/api/invoice/admin/applications', {
          params: { p: 1, page_size: 100 },
        }),
        API.get('/api/invoice/admin/records', {
          params: { p: 1, page_size: 100 },
        }),
      ]);
      if (applicationsRes.data.success) {
        setApplications(applicationsRes.data.data.items || []);
      }
      if (recordsRes.data.success) {
        setRecords(recordsRes.data.data.items || []);
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

  const reviewAction = async (id, action) => {
    const payload = {
      admin_remark: remark,
    };
    if (action === 'reject') {
      let rejectedReason = '';
      await new Promise((resolve) => {
        Modal.confirm({
          title: t('填写驳回原因'),
          content: (
            <Input
              placeholder={t('请输入驳回原因')}
              onChange={(value) => {
                rejectedReason = value;
              }}
            />
          ),
          onOk: resolve,
          onCancel: resolve,
        });
      });
      payload.rejected_reason = rejectedReason;
      if (!rejectedReason) {
        return;
      }
    }
    try {
      const res = await API.post(
        `/api/invoice/admin/applications/${id}/${action}`,
        payload,
      );
      if (res.data.success) {
        showSuccess(action === 'approve' ? t('申请已通过') : t('申请已驳回'));
        loadData();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error);
    }
  };

  const issueRecord = async () => {
    if (selectedApplicationIds.length === 0) {
      showError(t('请先选择申请单'));
      return;
    }
    setIssuing(true);
    try {
      const res = await API.post('/api/invoice/admin/records', {
        application_ids: selectedApplicationIds,
        invoice_no: invoiceNo,
        file_url: fileURL,
        remark,
      });
      if (res.data.success) {
        showSuccess(t('发票记录已生成'));
        setSelectedApplicationIds([]);
        setInvoiceNo('');
        setFileURL('');
        setRemark('');
        loadData();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error);
    } finally {
      setIssuing(false);
    }
  };

  const applicationColumns = useMemo(
    () => [
      {
        title: t('申请单号'),
        dataIndex: 'application_no',
      },
      {
        title: t('用户ID'),
        dataIndex: 'user_id',
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        render: (value) => <Tag color='blue'>{value}</Tag>,
      },
      {
        title: t('申请金额'),
        render: (_, record) => formatAmount(record.total_amount, record.currency),
      },
      {
        title: t('订单数'),
        render: (_, record) => record.items?.length || 0,
      },
      {
        title: t('提交时间'),
        render: (_, record) => formatTime(record.created_at),
      },
      {
        title: t('操作'),
        render: (_, record) => (
          <div className='flex gap-2'>
            <Button
              size='small'
              theme='solid'
              disabled={record.status !== 'pending_review'}
              onClick={() => reviewAction(record.id, 'approve')}
            >
              {t('通过')}
            </Button>
            <Button
              size='small'
              type='danger'
              theme='borderless'
              disabled={
                !['pending_review', 'approved'].includes(record.status)
              }
              onClick={() => reviewAction(record.id, 'reject')}
            >
              {t('驳回')}
            </Button>
          </div>
        ),
      },
    ],
    [t, remark],
  );

  const recordColumns = useMemo(
    () => [
      {
        title: t('发票号'),
        dataIndex: 'invoice_no',
      },
      {
        title: t('用户ID'),
        dataIndex: 'user_id',
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
      {
        title: t('操作'),
        render: (_, record) => (
          <Button
            size='small'
            type='danger'
            theme='borderless'
            disabled={record.status !== 'issued'}
            onClick={async () => {
              try {
                const res = await API.post(
                  `/api/invoice/admin/records/${record.id}/void`,
                  { remark: t('管理员作废') },
                );
                if (res.data.success) {
                  showSuccess(t('发票已作废'));
                  loadData();
                } else {
                  showError(res.data.message);
                }
              } catch (error) {
                showError(error);
              }
            }}
          >
            {t('作废')}
          </Button>
        ),
      },
    ],
    [t],
  );

  const rowSelection = {
    selectedRowKeys: selectedApplicationIds,
    getCheckboxProps: (record) => ({
      disabled: record.status !== 'approved',
      name: record.id,
    }),
    onChange: (keys) => setSelectedApplicationIds(keys),
  };

  return (
    <div className='mt-[60px] px-2'>
      <Card bordered={false}>
        <Typography.Title heading={4}>{t('发票管理')}</Typography.Title>
        <Typography.Text type='secondary'>
          {t('审核用户开票申请，并支持按用户合并生成正式发票记录。')}
        </Typography.Text>
      </Card>

      <Tabs className='mt-4' type='line'>
        <TabPane tab={t('申请单')} itemKey='applications'>
          <Banner
            type='warning'
            closeIcon={null}
            description={t('只有已通过审核的申请，才能被合并开票。')}
          />
          <Card className='mt-4' loading={loading}>
            <div className='mb-4 flex flex-wrap gap-3'>
              <Input
                style={{ width: 220 }}
                placeholder={t('发票号，可留空自动生成')}
                value={invoiceNo}
                onChange={setInvoiceNo}
              />
              <Input
                style={{ width: 280 }}
                placeholder={t('附件链接')}
                value={fileURL}
                onChange={setFileURL}
              />
              <Input
                style={{ width: 240 }}
                placeholder={t('备注')}
                value={remark}
                onChange={setRemark}
              />
              <Button theme='solid' loading={issuing} onClick={issueRecord}>
                {t('合并开票')}
              </Button>
              <Button onClick={loadData}>{t('刷新')}</Button>
            </div>
            <Table
              rowKey='id'
              columns={applicationColumns}
              dataSource={applications}
              rowSelection={rowSelection}
              pagination={false}
              empty={<Empty title={t('暂无申请单')} />}
            />
          </Card>
        </TabPane>

        <TabPane tab={t('发票记录')} itemKey='records'>
          <Card className='mt-4' loading={loading}>
            <Table
              rowKey='id'
              columns={recordColumns}
              dataSource={records}
              pagination={false}
              empty={<Empty title={t('暂无发票记录')} />}
            />
          </Card>
        </TabPane>
      </Tabs>
    </div>
  );
};

export default InvoiceAdminPage;
