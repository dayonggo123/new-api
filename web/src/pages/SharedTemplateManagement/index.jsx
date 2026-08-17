import React, { useState, useEffect, useCallback } from 'react';
import {
  Button,
  Card,
  Table,
  Modal,
  Tag,
  Space,
  Typography,
  Empty,
  Select,
} from '@douyinfe/semi-ui';
import { IconRefresh } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const { Text, Paragraph } = Typography;

const STATUS_MAP = {
  pending: { text: '待审核', color: 'warning' },
  approved: { text: '已通过', color: 'green' },
  rejected: { text: '已拒绝', color: 'red' },
};

const CATEGORY_MAP = {
  ecommerce: '电商',
  portrait: '人像',
  landscape: '风景',
  commercial: '商业',
  creative: '创意',
  other: '其他',
};

const PAGE_SIZE = 20;

export default function SharedTemplateManagement() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('pending');
  const [detailVisible, setDetailVisible] = useState(false);
  const [detailItem, setDetailItem] = useState(null);
  const [auditing, setAuditing] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const params = { page, page_size: PAGE_SIZE };
      let url;
      if (statusFilter === 'pending') {
        url = '/api/admin/shared-templates/pending';
      } else {
        url = '/api/admin/shared-templates/all';
        if (statusFilter) params.status = statusFilter;
      }
      const res = await API.get(url, { params });
      const { success, message, data: result } = res.data;
      if (success && result) {
        setData(result.list || []);
        setTotal(result.total || 0);
      } else {
        showError(message || t('加载失败'));
      }
    } catch (err) {
      showError(err.message);
    } finally {
      setLoading(false);
    }
  }, [page, statusFilter, t]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Reset page when filter changes
  const handleStatusFilterChange = (value) => {
    setStatusFilter(value);
    setPage(1);
  };

  const handleViewDetail = async (record) => {
    try {
      const res = await API.get(`/api/admin/shared-templates/${record.id}`);
      if (res.data.success) {
        setDetailItem(res.data.data);
        setDetailVisible(true);
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.message);
    }
  };

  const handleDelete = (record) => {
    Modal.confirm({
      title: t('删除模板'),
      content: t('确定删除模板「{name}」吗？删除后将从模板市场移除，且不可恢复。', { name: record.name || record.id }),
      type: 'warning',
      okText: t('删除'),
      cancelText: t('取消'),
      onOk: async () => {
        try {
          const res = await API.delete(`/api/admin/shared-templates/${record.id}`);
          if (res.data.success) {
            showSuccess(t('已删除'));
            loadData();
          } else {
            showError(res.data.message);
          }
        } catch (err) {
          showError(err.message);
        }
      },
    });
  };

  const handleAudit = async (record, action) => {
    let reason = '';
    if (action === 'reject') {
      // Ask for reason using Modal
      const reasonInput = await new Promise((resolve) => {
        let inputValue = '';
        Modal.info({
          title: t('拒绝原因'),
          content: (
            <div style={{ marginTop: 16 }}>
              <Text type='secondary' style={{ display: 'block', marginBottom: 8 }}>
                {t('请输入拒绝原因（最长 500 字符）')}
              </Text>
              <textarea
                rows={4}
                style={{ width: '100%', padding: 8, border: '1px solid var(--semi-color-border)', borderRadius: 4 }}
                placeholder={t('如：含版权素材，请修改后重新提交')}
                onChange={(e) => { inputValue = e.target.value; }}
              />
              <div style={{ marginTop: 12, textAlign: 'right' }}>
                <Button onClick={() => resolve(null)} style={{ marginRight: 8 }}>
                  {t('取消')}
                </Button>
                <Button type='primary' onClick={() => resolve(inputValue || '')}>
                  {t('确认拒绝')}
                </Button>
              </div>
            </div>
          ),
          icon: null,
          footer: null,
          width: 480,
        });
      });
      if (reasonInput === null) return; // cancelled
      reason = reasonInput;
    }

    setAuditing(true);
    try {
      const res = await API.post(`/api/admin/shared-templates/${record.id}/audit`, {
        action,
        reason,
      });
      if (res.data.success) {
        showSuccess(action === 'approve' ? t('已通过') : t('已拒绝'));
        loadData();
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.message);
    } finally {
      setAuditing(false);
    }
  };

  const formatTime = (ts) => {
    if (!ts) return '-';
    return new Date(ts * 1000).toLocaleString('zh-CN');
  };

  const renderStatus = (status) => {
    const cfg = STATUS_MAP[status] || { text: status, color: 'default' };
    return <Tag color={cfg.color}>{cfg.text}</Tag>;
  };

  const columns = [
    {
      title: t('名称'),
      dataIndex: 'name',
      key: 'name',
      width: 200,
      render: (text, record) => (
        <Button
          type='tertiary'
          size='small'
          style={{ textAlign: 'left', padding: 0, maxWidth: 180 }}
          onClick={() => handleViewDetail(record)}
        >
          <Text ellipsis={{ showTooltip: true }}>{text || '-'}</Text>
        </Button>
      ),
    },
    {
      title: t('分类'),
      dataIndex: 'category',
      key: 'category',
      width: 80,
      render: (v) => CATEGORY_MAP[v] || v || '-',
    },
    {
      title: t('作者'),
      dataIndex: 'authorName',
      key: 'authorName',
      width: 120,
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (v) => renderStatus(v),
    },
    {
      title: t('使用次数'),
      dataIndex: 'useCount',
      key: 'useCount',
      width: 80,
    },
    {
      title: t('提交时间'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      render: (v) => formatTime(v),
    },
    {
      title: t('操作'),
      key: 'action',
      width: 280,
      fixed: 'right',
      render: (_, record) => (
        <Space>
          <Button type='tertiary' size='small' onClick={() => handleViewDetail(record)}>
            {t('查看')}
          </Button>
          {record.status === 'pending' && (
            <>
              <Button
                type='primary'
                size='small'
                loading={auditing}
                onClick={() => handleAudit(record, 'approve')}
              >
                {t('通过')}
              </Button>
              <Button
                type='danger'
                size='small'
                loading={auditing}
                onClick={() => handleAudit(record, 'reject')}
              >
                {t('拒绝')}
              </Button>
            </>
          )}
          <Button type='danger' theme='borderless' size='small' onClick={() => handleDelete(record)}>
            {t('删除')}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <div style={{ marginBottom: 16, padding: 12, background: '#f0f9ff', borderRadius: 8, borderLeft: '4px solid #0ea5e9' }}>
        <Text type='secondary' style={{ fontSize: 13, lineHeight: 1.6 }}>
          <strong>{t('功能介绍：')}</strong>{t('管理用户提交的工作流模板分享，审核通过后将在模板市场展示')}<br />
          <strong>{t('审核流程：')}</strong>{t('用户提交 → 待审核 → 管理员通过/拒绝 → 通过后即上线')}
        </Text>
      </div>
      <Card
        title={t('模板审核管理')}
        headerExtraContent={
          <Space>
            <Select
              value={statusFilter}
              onChange={handleStatusFilterChange}
              style={{ width: 120 }}
            >
              <Select.Option value='pending'>{t('待审核')}</Select.Option>
              <Select.Option value='approved'>{t('已通过')}</Select.Option>
              <Select.Option value='rejected'>{t('已拒绝')}</Select.Option>
              <Select.Option value=''>{t('全��')}</Select.Option>
            </Select>
            <Button icon={<IconRefresh />} onClick={loadData}>
              {t('刷新')}
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={data}
          loading={loading}
          rowKey='id'
          pagination={{
            currentPage: page,
            pageSize: PAGE_SIZE,
            total,
            onPageChange: (p) => setPage(p),
            showSizeChanger: false,
          }}
          empty={<Empty description={t('暂无数据')} />}
        />
      </Card>

      {/* Detail Modal */}
      <Modal
        title={t('模板详情')}
        visible={detailVisible}
        onCancel={() => { setDetailVisible(false); setDetailItem(null); }}
        footer={null}
        centered
        width={720}
        bodyStyle={{ maxHeight: '70vh', overflow: 'auto' }}
      >
        {detailItem && (
          <div>
            <div style={{ marginBottom: 16 }}>
              <Text strong>{t('名称：')}</Text>
              <Text>{detailItem.name || '-'}</Text>
            </div>
            <div style={{ marginBottom: 16 }}>
              <Text strong>{t('分类：')}</Text>
              <Text>{CATEGORY_MAP[detailItem.category] || detailItem.category || '-'}</Text>
              <Text style={{ marginLeft: 24 }}><strong>{t('状态：')}</strong></Text>
              {renderStatus(detailItem.status)}
            </div>
            <div style={{ marginBottom: 16 }}>
              <Text strong>{t('作者：')}</Text>
              <Text>{detailItem.authorName || '-'} (ID: {detailItem.authorId})</Text>
            </div>
            {detailItem.description && (
              <div style={{ marginBottom: 16 }}>
                <Text strong>{t('描述：')}</Text>
                <Paragraph>{detailItem.description}</Paragraph>
              </div>
            )}
            <div style={{ marginBottom: 8 }}>
              <Text strong>{t('版本：')}</Text>
              <Text>{detailItem.planVersion || '-'}</Text>
              {detailItem.appMinVersion && (
                <Text style={{ marginLeft: 24 }}>
                  <strong>{t('最低兼容：')}</strong>{detailItem.appMinVersion}
                </Text>
              )}
            </div>
            {detailItem.rejectReason && (
              <div style={{ marginBottom: 16, padding: 8, background: '#fff3f3', borderRadius: 4 }}>
                <Text strong type='danger'>{t('拒绝原因：')}</Text>
                <Text type='danger'>{detailItem.rejectReason}</Text>
              </div>
            )}
            <div style={{ marginBottom: 8 }}>
              <Text strong>{t('执行计划 (planJson)：')}</Text>
            </div>
            <pre style={{
              background: '#f5f5f5',
              padding: 12,
              borderRadius: 4,
              maxHeight: 400,
              overflow: 'auto',
              fontSize: 12,
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-all',
            }}>
              {(() => {
                try {
                  return JSON.stringify(JSON.parse(detailItem.planJson), null, 2);
                } catch {
                  return detailItem.planJson;
                }
              })()}
            </pre>
            <div style={{ marginTop: 16, textAlign: 'right' }}>
              <Space>
                <Button
                  type='danger'
                  theme='borderless'
                  loading={auditing}
                  onClick={() => { handleDelete(detailItem); setDetailVisible(false); }}
                >
                  {t('删除')}
                </Button>
                {detailItem.status === 'pending' && (
                  <>
                    <Button
                      type='primary'
                      loading={auditing}
                      onClick={() => { handleAudit(detailItem, 'approve'); setDetailVisible(false); }}
                    >
                      {t('通过')}
                    </Button>
                    <Button
                      type='danger'
                      loading={auditing}
                      onClick={() => { handleAudit(detailItem, 'reject'); setDetailVisible(false); }}
                    >
                      {t('拒绝')}
                    </Button>
                  </>
                )}
              </Space>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
