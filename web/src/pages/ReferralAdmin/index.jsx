import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Empty,
  Input,
  SideSheet,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError } from '../../helpers';

const DEFAULT_PAGE_SIZE = 20;

const formatAmount = (amount) => Number(amount || 0).toFixed(2);
const formatTime = (timestamp) => {
  if (!timestamp) return '-';
  return new Date(timestamp * 1000).toLocaleString();
};

const StatCard = ({ title, value, subtitle }) => (
  <Card className='min-w-[220px] flex-1'>
    <Typography.Text type='tertiary'>{title}</Typography.Text>
    <Typography.Title heading={3} style={{ marginTop: 12, marginBottom: 8 }}>
      {value}
    </Typography.Title>
    <Typography.Text type='secondary'>{subtitle}</Typography.Text>
  </Card>
);

const ReferralAdminPage = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [keyword, setKeyword] = useState('');
  const [summary, setSummary] = useState(null);
  const [relations, setRelations] = useState([]);
  const [relationPage, setRelationPage] = useState(1);
  const [relationPageSize, setRelationPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [relationTotal, setRelationTotal] = useState(0);
  const [detail, setDetail] = useState(null);
  const [detailVisible, setDetailVisible] = useState(false);

  const loadData = async ({
    nextKeyword = keyword,
    nextPage = relationPage,
    nextPageSize = relationPageSize,
  } = {}) => {
    setLoading(true);
    try {
      const [summaryRes, relationsRes] = await Promise.all([
        API.get('/api/referral/admin/summary'),
        API.get('/api/referral/admin/relations', {
          params: {
            p: nextPage,
            page_size: nextPageSize,
            keyword: nextKeyword || undefined,
          },
        }),
      ]);
      if (summaryRes.data.success) {
        setSummary(summaryRes.data.data);
      }
      if (relationsRes.data.success) {
        const payload = relationsRes.data.data || {};
        setRelations(payload.items || []);
        setRelationTotal(payload.total || 0);
        setRelationPage(payload.page || nextPage);
        setRelationPageSize(payload.page_size || nextPageSize);
      }
    } catch (error) {
      showError(error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData({ nextKeyword: '' });
  }, []);

  const relationColumns = useMemo(
    () => [
      {
        title: t('邀请人'),
        render: (_, record) =>
          `${record.inviter_username || '-'} (#${record.inviter_id || '-'})`,
      },
      {
        title: t('被邀请人'),
        render: (_, record) =>
          `${record.invitee_username || '-'} (#${record.invitee_id || '-'})`,
      },
      {
        title: t('邀请人数'),
        dataIndex: 'inviter_aff_count',
      },
      {
        title: t('累计返利额度'),
        dataIndex: 'inviter_reward_quota',
      },
      {
        title: t('被邀请人充值'),
        render: (_, record) => formatAmount(record.invitee_topup_amount),
      },
      {
        title: t('被邀请人消费'),
        dataIndex: 'invitee_used_quota',
      },
      {
        title: t('注册IP'),
        render: (_, record) =>
          `${record.inviter_register_ip || '-'} / ${record.invitee_register_ip || '-'}`,
      },
      {
        title: t('最近登录IP'),
        render: (_, record) =>
          `${record.inviter_last_login_ip || '-'} / ${record.invitee_last_login_ip || '-'}`,
      },
      {
        title: t('最近支付IP'),
        render: (_, record) =>
          `${record.inviter_last_topup_ip || '-'} / ${record.invitee_last_topup_ip || '-'}`,
      },
      {
        title: t('风险标签'),
        render: (_, record) =>
          record.risk_tags?.length ? (
            <div className='flex flex-wrap gap-2'>
              {record.risk_tags.map((tag) => (
                <Tag key={tag} color='red'>
                  {tag}
                </Tag>
              ))}
            </div>
          ) : (
            <Tag color='green'>{t('未命中')}</Tag>
          ),
      },
      {
        title: t('操作'),
        render: (_, record) => (
          <Button
            size='small'
            theme='borderless'
            onClick={async () => {
              try {
                const res = await API.get(
                  `/api/referral/admin/relations/${record.invitee_id}`,
                );
                if (res.data.success) {
                  setDetail(res.data.data);
                  setDetailVisible(true);
                } else {
                  showError(res.data.message);
                }
              } catch (error) {
                showError(error);
              }
            }}
          >
            {t('查看详情')}
          </Button>
        ),
      },
    ],
    [t],
  );

  const topupColumns = useMemo(
    () => [
      {
        title: t('订单号'),
        dataIndex: 'trade_no',
      },
      {
        title: t('支付方式'),
        dataIndex: 'payment_method',
      },
      {
        title: t('支付金额'),
        render: (_, record) =>
          `${Number(record.paid_amount || record.money || 0).toFixed(2)} ${record.paid_currency || ''}`.trim(),
      },
      {
        title: t('IP'),
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

  return (
    <div className='mt-[60px] px-2'>
      <Card bordered={false}>
        <Typography.Title heading={4}>{t('邀请管理')}</Typography.Title>
        <Typography.Text type='secondary'>
          {t('查看邀请关系、充值消费与 IP 风险，用于识别大号邀请小号套利行为。')}
        </Typography.Text>
      </Card>

      <div className='mt-4 flex flex-wrap gap-4'>
        <StatCard
          title={t('邀请关系')}
          value={summary?.total_relations || 0}
          subtitle={t('当前存在 inviter_id 关系的用户数')}
        />
        <StatCard
          title={t('有充值的被邀请人')}
          value={summary?.invitee_with_topup_count || 0}
          subtitle={t('已经产生支付行为的被邀请账号')}
        />
        <StatCard
          title={t('被邀请人总充值')}
          value={formatAmount(summary?.total_invitee_topup || 0)}
          subtitle={t('用于识别返利套利规模')}
        />
        <StatCard
          title={t('同 IP 风险')}
          value={`${summary?.same_register_ip_count || 0} / ${summary?.same_login_ip_count || 0} / ${summary?.same_topup_ip_count || 0}`}
          subtitle={t('注册 / 登录 / 支付 三类风险')}
        />
      </div>

      <Card className='mt-4' loading={loading}>
        <div className='mb-4 flex flex-wrap gap-3'>
          <Input
            style={{ width: 280 }}
            placeholder={t('搜索邀请人、被邀请人、邮箱')}
            value={keyword}
            onChange={setKeyword}
          />
          <Button
            theme='solid'
            onClick={() => loadData({ nextKeyword: keyword, nextPage: 1 })}
          >
            {t('搜索')}
          </Button>
          <Button onClick={() => loadData({ nextKeyword: keyword })}>
            {t('刷新')}
          </Button>
        </div>
        <Table
          rowKey='invitee_id'
          columns={relationColumns}
          dataSource={relations}
          pagination={{
            currentPage: relationPage,
            pageSize: relationPageSize,
            total: relationTotal,
            showSizeChanger: true,
            pageSizeOpts: [10, 20, 50, 100],
            onPageChange: (page) => {
              loadData({ nextKeyword: keyword, nextPage: page });
            },
            onPageSizeChange: (pageSize) => {
              loadData({
                nextKeyword: keyword,
                nextPage: 1,
                nextPageSize: pageSize,
              });
            },
          }}
          empty={<Empty title={t('暂无邀请关系')} />}
        />
      </Card>

      <SideSheet
        title={t('邀请详情')}
        visible={detailVisible}
        onCancel={() => setDetailVisible(false)}
        width={920}
      >
        {detail ? (
          <>
            <Card title={t('关系概览')} className='mb-4'>
              <div className='grid gap-3'>
                <Typography.Text>
                  {t('邀请人')}: {detail.relation.inviter_username} (
                  {detail.relation.inviter_id})
                </Typography.Text>
                <Typography.Text>
                  {t('被邀请人')}: {detail.relation.invitee_username} (
                  {detail.relation.invitee_id})
                </Typography.Text>
                <Typography.Text>
                  {t('风险标签')}:{' '}
                  {detail.relation.risk_tags?.length
                    ? detail.relation.risk_tags.join(' / ')
                    : t('未命中')}
                </Typography.Text>
              </div>
            </Card>
            <Card title={t('被邀请人充值记录')}>
              <Table
                rowKey='id'
                columns={topupColumns}
                dataSource={detail.topups || []}
                pagination={false}
                empty={<Empty title={t('暂无充值记录')} />}
              />
            </Card>
          </>
        ) : (
          <Empty title={t('暂无详情')} />
        )}
      </SideSheet>
    </div>
  );
};

export default ReferralAdminPage;
