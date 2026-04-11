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
import {
  copyInvoiceApplicationInfo,
  formatAmount,
  formatTime,
  InvoiceApplicationDetails,
  renderInvoiceProfileSummary,
  renderInvoiceStatusTag,
} from '../Invoice/invoiceShared';

const { TabPane } = Tabs;

const InvoiceAdminPage = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [applications, setApplications] = useState([]);
  const [records, setRecords] = useState([]);
  const [remark, setRemark] = useState('');
  const [detailApplication, setDetailApplication] = useState(null);

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
      let confirmed = false;
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
          onOk: () => {
            confirmed = true;
            resolve();
          },
          onCancel: resolve,
        });
      });
      if (!confirmed || !rejectedReason) {
        return;
      }
      payload.rejected_reason = rejectedReason;
    }
    try {
      const res = await API.post(
        `/api/invoice/admin/applications/${id}/${action}`,
        payload,
      );
      if (res.data.success) {
        showSuccess(
          action === 'approve'
            ? t('申请已通过，并已直接标记为开票成功')
            : t('申请已驳回'),
        );
        loadData();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error);
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
        render: (value) => renderInvoiceStatusTag(value, t),
      },
      {
        title: t('开票信息'),
        render: (_, record) => renderInvoiceProfileSummary(record, t),
      },
      {
        title: t('申请金额'),
        render: (_, record) =>
          formatAmount(record.total_amount, record.currency),
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
              theme='solid'
              disabled={record.status !== 'pending_review'}
              onClick={() => reviewAction(record.id, 'approve')}
            >
              {t('通过并完成开票')}
            </Button>
            <Button
              size='small'
              type='danger'
              theme='borderless'
              disabled={record.status !== 'pending_review'}
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

  return (
    <div className='mt-[60px] px-2'>
      <Card bordered={false}>
        <Typography.Title heading={4}>{t('发票管理')}</Typography.Title>
        <Typography.Text type='secondary'>
          {t(
            '审核用户开票申请。抬头、税号、邮箱为必填项，通过后系统会直接记为开票成功，不再保留平台内发票附件存档。',
          )}
        </Typography.Text>
      </Card>

      <Tabs className='mt-4' type='line'>
        <TabPane tab={t('申请单')} itemKey='applications'>
          <Banner
            type='warning'
            closeIcon={null}
            description={t(
              '审批通过后会直接进入开票成功，并生成一条开票记录。平台不再提供单独的发票附件存档流程。',
            )}
          />
          <Card className='mt-4' loading={loading}>
            <div className='mb-4 flex flex-wrap gap-3'>
              <Input
                style={{ width: 280 }}
                placeholder={t('审核备注')}
                value={remark}
                onChange={setRemark}
              />
              <Button onClick={loadData}>{t('刷新')}</Button>
            </div>
            <Table
              rowKey='id'
              columns={applicationColumns}
              dataSource={applications}
              pagination={false}
              empty={<Empty title={t('暂无申请单')} />}
            />
          </Card>
        </TabPane>

        <TabPane tab={t('开票记录')} itemKey='records'>
          <Card className='mt-4' loading={loading}>
            <Table
              rowKey='id'
              columns={recordColumns}
              dataSource={records}
              pagination={false}
              empty={<Empty title={t('暂无开票记录')} />}
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
        <InvoiceApplicationDetails
          application={detailApplication}
          t={t}
          showUserId
        />
      </Modal>
    </div>
  );
};

export default InvoiceAdminPage;
