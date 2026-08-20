import React, { useEffect, useRef, useState } from 'react';
import { Empty, Modal, Table, Tabs, Tag } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, renderQuota, showError } from '../../../../helpers';

const PAGE_SIZE = 10;

const PAYMENT_METHOD_MAP = {
  stripe: 'Stripe',
  alipay: '支付宝',
  epay: '易支付',
  yizhifu: '一支付',
  waffo: 'Waffo',
  creem: 'Creem',
  manual: '手动',
};

const TOPUP_STATUS_MAP = {
  pending: { text: '待支付', color: 'warning' },
  success: { text: '成功', color: 'green' },
  expired: { text: '已过期', color: 'red' },
  cancelled: { text: '已取消', color: 'grey' },
};

const formatTime = (ts) => {
  if (!ts) return '-';
  return new Date(ts * 1000).toLocaleString('zh-CN');
};

const UserFinanceModal = ({ visible, onCancel, user }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [topups, setTopups] = useState([]);
  const [topupTotal, setTopupTotal] = useState(0);
  const [topupPage, setTopupPage] = useState(1);
  const [logs, setLogs] = useState([]);
  const [logTotal, setLogTotal] = useState(0);
  const [logPage, setLogPage] = useState(1);

  const mountedRef = useRef(false);
  const fetchedRef = useRef(false);

  const safeSetState = (setter) => (value) => {
    if (mountedRef.current) {
      setter(value);
    }
  };

  const loadTopups = async (page = 1) => {
    if (!user?.id) return;
    safeSetState(setLoading)(true);
    try {
      const res = await API.get(`/api/admin/users/${user.id}/topups`, {
        params: { page, page_size: PAGE_SIZE },
      });
      if (res.data?.success) {
        safeSetState(setTopups)(res.data.data?.items || []);
        safeSetState(setTopupTotal)(res.data.data?.total || 0);
        safeSetState(setTopupPage)(page);
      } else {
        const msg = res.data?.message || t('请求失败') || '请求失败';
        showError(msg);
      }
    } catch (e) {
      console.error('[UserFinanceModal] loadTopups error:', e);
      const msg = e?.message || e?.toString?.() || t('请求失败') || '请求失败';
      showError(msg);
    } finally {
      safeSetState(setLoading)(false);
    }
  };

  const loadLogs = async (page = 1) => {
    if (!user?.id) return;
    safeSetState(setLoading)(true);
    try {
      const res = await API.get(`/api/admin/users/${user.id}/usage-logs`, {
        params: { page, page_size: PAGE_SIZE },
      });
      if (res.data?.success) {
        safeSetState(setLogs)(res.data.data?.items || []);
        safeSetState(setLogTotal)(res.data.data?.total || 0);
        safeSetState(setLogPage)(page);
      } else {
        const msg = res.data?.message || t('请求失败') || '请求失败';
        showError(msg);
      }
    } catch (e) {
      console.error('[UserFinanceModal] loadLogs error:', e);
      const msg = e?.message || e?.toString?.() || t('请求失败') || '请求失败';
      showError(msg);
    } finally {
      safeSetState(setLoading)(false);
    }
  };

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  useEffect(() => {
    if (!visible || !user?.id) {
      fetchedRef.current = false;
      return;
    }
    if (fetchedRef.current) return;
    fetchedRef.current = true;
    loadTopups(1);
    loadLogs(1);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible, user?.id]);

  const renderPaymentMethod = (v) => {
    if (!v) return <Tag size='small' color='white'>-</Tag>;
    return <Tag size='small' color='blue'>{PAYMENT_METHOD_MAP[v] || v}</Tag>;
  };

  const renderTopUpStatus = (v) => {
    const cfg = TOPUP_STATUS_MAP[v] || { text: v, color: 'white' };
    return <Tag size='small' color={cfg.color}>{cfg.text}</Tag>;
  };

  const topupColumns = [
    { title: t('时间'), dataIndex: 'create_time', width: 170, render: formatTime },
    { title: t('充值渠道'), dataIndex: 'payment_method', width: 110, render: renderPaymentMethod },
    {
      title: t('金额'),
      dataIndex: 'money',
      width: 110,
      render: (v, record) => `${v ?? 0} 元`,
    },
    {
      title: t('充值额度'),
      dataIndex: 'amount',
      width: 120,
      render: (v) => renderQuota(v || 0),
    },
    { title: t('状态'), dataIndex: 'status', width: 100, render: renderTopUpStatus },
    { title: t('订单号'), dataIndex: 'trade_no', render: (v) => <span style={{ wordBreak: 'break-all' }}>{v || '-'}</span> },
  ];

  const logColumns = [
    { title: t('时间'), dataIndex: 'created_at', width: 170, render: formatTime },
    { title: t('模型'), dataIndex: 'model_name', render: (v) => v || '-' },
    {
      title: t('消耗额度'),
      dataIndex: 'quota',
      width: 130,
      render: (v) => <span style={{ color: 'var(--semi-color-danger)' }}>{renderQuota(v || 0)}</span>,
    },
    { title: t('Token'), dataIndex: 'token_name', width: 140, render: (v) => v || '-' },
    { title: t('渠道'), dataIndex: 'channel_name', width: 120, render: (v) => v || '-' },
  ];

  return (
    <Modal
      title={`${t('财务记录')} - ${user?.username || user?.id || ''}`}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      centered
      width={860}
      bodyStyle={{ maxHeight: '70vh', overflow: 'auto' }}
    >
      <Tabs type='line'>
        <Tabs.TabPane tab={t('充值记录')} itemKey='topups'>
          <Table
            columns={topupColumns}
            dataSource={topups}
            rowKey='id'
            loading={loading}
            pagination={{
              currentPage: topupPage,
              pageSize: PAGE_SIZE,
              total: topupTotal,
              onPageChange: loadTopups,
              showSizeChanger: false,
            }}
            empty={<Empty description={t('暂无充值记录')} />}
            size='small'
          />
        </Tabs.TabPane>
        <Tabs.TabPane tab={t('消费记录')} itemKey='logs'>
          <Table
            columns={logColumns}
            dataSource={logs}
            rowKey='id'
            loading={loading}
            pagination={{
              currentPage: logPage,
              pageSize: PAGE_SIZE,
              total: logTotal,
              onPageChange: loadLogs,
              showSizeChanger: false,
            }}
            empty={<Empty description={t('暂无消费记录')} />}
            size='small'
          />
        </Tabs.TabPane>
      </Tabs>
    </Modal>
  );
};

export default UserFinanceModal;
