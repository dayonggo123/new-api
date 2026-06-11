/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

SEO Center — 一站式 SEO 自动化中心
包含：关键词研究、内容生成、监控仪表盘
*/

import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Button,
  Card,
  Form,
  Input,
  Tabs,
  Tag,
  Spin,
  Typography,
  Table,
  Progress,
  Row,
  Col,
  Empty,
  Space,
  Descriptions,
  Badge,
} from '@douyinfe/semi-ui';
import {
  IconSearch,
  IconBolt,
  IconRefresh,
  IconEyeOpened,
  IconLanguage,
  IconTickCircle,
  IconAlertTriangle,
  IconInfoCircle,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const { Title, Text } = Typography;
const { TabPane } = Tabs;

// 简单统计卡片（Semi UI 无 Statistic 组件，用 Typography 模拟）
const StatCard = ({ title, value, suffix, precision }) => (
  <div style={{ textAlign: 'center' }}>
    <Title heading={2} style={{ margin: 0 }}>
      {precision !== undefined ? Number(value).toFixed(precision) : value}
      {suffix}
    </Title>
    <Text type="secondary" size="small">{title}</Text>
  </div>
);

// ==================== Tab 1: Keyword Research ====================

const KeywordResearchTab = ({ onAddKeyword }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState(null);
  const [templates, setTemplates] = useState([]);
  const [formApi, setFormApi] = useState(null);

  useEffect(() => {
    loadTemplates();
  }, []);

  const loadTemplates = async () => {
    try {
      const res = await API.get('/api/admin/seo/research/templates');
      if (res.data.success) {
        setTemplates(res.data.data || []);
      }
    } catch (e) {
      console.error('load templates failed', e);
    }
  };

  const handleResearch = async (values) => {
    if (!values.seed_keyword) {
      showError('请输入研究主题');
      return;
    }
    setLoading(true);
    try {
      const res = await API.post('/api/admin/seo/research', {
        seed_keyword: values.seed_keyword,
        language: values.language || 'en',
      });
      if (res.data.success) {
        setResult(res.data.data);
        showSuccess('关键词研究完成');
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e.message);
    }
    setLoading(false);
  };

  const getDifficultyColor = (d) => {
    switch (d) {
      case 'low': return 'green';
      case 'medium': return 'orange';
      case 'high': return 'red';
      default: return 'grey';
    }
  };

  const getIntentLabel = (i) => {
    const map = {
      informational: '信息型',
      navigational: '导航型',
      transactional: '交易型',
      commercial: '商业调查型',
    };
    return map[i] || i;
  };

  const columns = [
    { title: '关键词', dataIndex: 'keyword', width: 220 },
    { title: '月搜索量', dataIndex: 'search_volume', width: 90 },
    { title: '意图', dataIndex: 'intent', render: (v) => <Tag size="small">{getIntentLabel(v)}</Tag>, width: 90 },
    { title: '难度', dataIndex: 'difficulty', render: (v) => <Tag color={getDifficultyColor(v)} size="small">{v}</Tag>, width: 70 },
    { title: '商业价值', dataIndex: 'business_value', width: 70 },
    { title: 'ROI', dataIndex: 'roi_score', render: (v) => <Progress percent={v} size="small" showInfo={true} />, width: 100 },
    { title: '建议URL', dataIndex: 'suggested_url', width: 180 },
    {
      title: '操作',
      key: 'action',
      width: 110,
      render: (_, record) => (
        <Button
          type="tertiary"
          size="small"
          onClick={() => onAddKeyword && onAddKeyword(record.keyword)}
        >
          添加到生成
        </Button>
      ),
    },
  ];

  return (
    <div>
      <Card style={{ marginBottom: 16 }}>
        <Form layout="horizontal" onSubmit={handleResearch} getFormApi={setFormApi}>
          <Row gutter={16} align="middle">
            <Col span={12}>
              <Form.Input
                field="seed_keyword"
                label="研究主题"
                placeholder="输入关键词主题，如：AI video prompt library"
                rules={[{ required: true }]}
              />
            </Col>
            <Col span={6}>
              <Form.Select field="language" label="语言" initValue="en" style={{ width: '100%' }}>
                <Form.Select.Option value="en">English</Form.Select.Option>
                <Form.Select.Option value="ja">日本語</Form.Select.Option>
                <Form.Select.Option value="ko">한국어</Form.Select.Option>
                <Form.Select.Option value="es">Español</Form.Select.Option>
                <Form.Select.Option value="de">Deutsch</Form.Select.Option>
                <Form.Select.Option value="fr">Français</Form.Select.Option>
                <Form.Select.Option value="zh">中文</Form.Select.Option>
              </Form.Select>
            </Col>
            <Col span={6}>
              <Button type="primary" htmlType="submit" icon={<IconSearch />} loading={loading} style={{ marginTop: 28 }}>
                开始研究
              </Button>
            </Col>
          </Row>
        </Form>

        {templates.length > 0 && (
          <div style={{ marginTop: 12 }}>
            <Text type="secondary">快速模板：</Text>
            <Space wrap>
              {templates.map((tmpl) => (
                <Tag
                  key={tmpl.id}
                  style={{ cursor: 'pointer' }}
                  onClick={() => {
                    if (formApi) {
                      formApi.setValue('seed_keyword', tmpl.seed_keyword);
                    }
                  }}
                >
                  {tmpl.name}
                </Tag>
              ))}
            </Space>
          </div>
        )}
      </Card>

      {loading && (
        <Card style={{ textAlign: 'center', padding: 40 }}>
          <Spin size="large" tip={<span style={{ whiteSpace: 'nowrap' }}>AI 正在进行关键词研究，请稍候...</span>} />
        </Card>
      )}

      {result && !loading && (
        <>
          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={6}>
              <Card>
                <StatCard title="种子词" value={result.seed_keywords?.length || 0} />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <StatCard title="扩展词" value={result.extended_keywords?.length || 0} />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <StatCard title="高 ROI 词" value={result.high_roi_keywords?.length || 0} />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <StatCard title="主题簇" value={result.topic_clusters?.length || 0} />
              </Card>
            </Col>
          </Row>

          <Card title="高 ROI 关键词" style={{ marginBottom: 16 }}>
            <Table
              columns={columns}
              dataSource={result.high_roi_keywords || []}
              pagination={{ pageSize: 10 }}
              size="small"
            />
          </Card>

          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={12}>
              <Card title="种子词">
                <Table
                  columns={columns}
                  dataSource={result.seed_keywords || []}
                  pagination={{ pageSize: 5 }}
                  size="small"
                />
              </Card>
            </Col>
            <Col span={12}>
              <Card title="扩展词">
                <Table
                  columns={columns}
                  dataSource={result.extended_keywords || []}
                  pagination={{ pageSize: 5 }}
                  size="small"
                />
              </Card>
            </Col>
          </Row>

          <Card title="主题聚类" style={{ marginBottom: 16 }}>
            <Row gutter={16}>
              {result.topic_clusters?.map((cluster, idx) => (
                <Col span={12} key={idx} style={{ marginBottom: 12 }}>
                  <Card
                    size="small"
                    title={cluster.name}
                    extra={
                      <Space>
                        <Button
                          type="tertiary"
                          size="small"
                          onClick={() => onAddKeyword && onAddKeyword(cluster.pillar_keyword)}
                        >
                          添加 Pillar
                        </Button>
                        <Tag color={cluster.priority === 'P0' ? 'red' : cluster.priority === 'P1' ? 'orange' : 'grey'}>{cluster.priority}</Tag>
                      </Space>
                    }
                  >
                    <Descriptions size="small" layout="vertical">
                      <Descriptions.Item itemKey="Pillar 关键词">
                        {cluster.pillar_keyword} ({cluster.pillar_volume}/月)
                      </Descriptions.Item>
                      <Descriptions.Item itemKey="簇内关键词">
                        {cluster.cluster_keywords?.join(', ')}
                        {cluster.cluster_keywords?.length > 0 && (
                          <Button
                            type="tertiary"
                            size="small"
                            style={{ marginLeft: 8 }}
                            onClick={() => {
                              if (onAddKeyword) {
                                cluster.cluster_keywords.forEach((kw) => onAddKeyword(kw));
                              }
                            }}
                          >
                            添加全部
                          </Button>
                        )}
                      </Descriptions.Item>
                      <Descriptions.Item itemKey="内容类型">{cluster.content_type}</Descriptions.Item>
                    </Descriptions>
                  </Card>
                </Col>
              ))}
            </Row>
          </Card>

          <Card title="内容缺口">
            <Table
              columns={[
                { title: '关键词', dataIndex: 'keyword', width: 180 },
                { title: '搜索量', dataIndex: 'volume', width: 80 },
                { title: '竞品覆盖', dataIndex: 'competitors', width: 100 },
                { title: '缺口类型', dataIndex: 'gap_type', render: (v) => <Tag size="small">{v}</Tag>, width: 90 },
                { title: '优先级', dataIndex: 'priority', render: (v) => <Tag color={v === 'P0' ? 'red' : v === 'P1' ? 'orange' : 'grey'}>{v}</Tag>, width: 70 },
                { title: '建议行动', dataIndex: 'suggested_action' },
                {
                  title: '操作',
                  key: 'action',
                  width: 100,
                  render: (_, record) => (
                    <Button
                      type="tertiary"
                      size="small"
                      onClick={() => onAddKeyword && onAddKeyword(record.keyword)}
                    >
                      添加到生成
                    </Button>
                  ),
                },
              ]}
              dataSource={result.content_gaps || []}
              pagination={false}
              size="small"
            />
          </Card>
        </>
      )}
    </div>
  );
};

// ==================== Tab 2: Content Generator ====================

const ContentGeneratorTab = ({ generatorKeywords }) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [queue, setQueue] = useState([]);
  const [genFormApi, setGenFormApi] = useState(null);

  // 当从关键词研究页面添加关键词时，自动填充到输入框
  useEffect(() => {
    if (genFormApi && generatorKeywords) {
      const current = genFormApi.getValue('keywords') || '';
      const newValue = current ? `${current}, ${generatorKeywords}` : generatorKeywords;
      genFormApi.setValue('keywords', newValue);
    }
  }, [generatorKeywords, genFormApi]);

  const handleGenerate = async (values) => {
    if (!values.keywords) {
      showError('请输入关键词');
      return;
    }
    const keywords = values.keywords.split(/[,，]/).map((k) => k.trim()).filter(Boolean);
    if (keywords.length === 0) {
      showError('请输入有效关键词');
      return;
    }

    const options = values.options || [];
    const auto_seo = options.includes('auto_seo');
    const auto_geo = options.includes('auto_geo');
    const auto_translate = options.includes('auto_translate');
    const auto_publish = options.includes('auto_publish');

    const task = {
      id: Date.now(),
      keywords: keywords,
      type: values.type || 'article',
      status: 'generating',
      progress: 0,
      message: '生成中...',
    };
    setQueue((prev) => [...prev, task]);
    setLoading(true);

    try {
      const res = await API.post('/api/admin/content/generate', {
        type: values.type || 'article',
        keywords: keywords,
        language: values.language || 'en',
        auto_seo: auto_seo,
        auto_geo: auto_geo,
        auto_translate: auto_translate,
        auto_publish: auto_publish,
      });

      if (res.data.success) {
        const data = res.data.data;
        setQueue((prev) =>
          prev.map((t) =>
            t.id === task.id
              ? { ...t, status: 'completed', progress: 100, message: `完成！ID: ${data.record_id}`, recordId: data.record_id }
              : t
          )
        );
        showSuccess('内容生成完成');
      } else {
        setQueue((prev) =>
          prev.map((t) =>
            t.id === task.id ? { ...t, status: 'failed', message: res.data.message } : t
          )
        );
        showError(res.data.message);
      }
    } catch (e) {
      setQueue((prev) =>
        prev.map((t) =>
          t.id === task.id ? { ...t, status: 'failed', message: e.message } : t
        )
      );
      showError(e.message);
    }
    setLoading(false);
  };

  const getStatusBadge = (status) => {
    switch (status) {
      case 'completed': return <Badge type="success" text="完成" />;
      case 'failed': return <Badge type="danger" text="失败" />;
      case 'generating': return <Badge type="warning" text="生成中" />;
      default: return <Badge type="default" text={status} />;
    }
  };

  return (
    <div>
      <Card style={{ marginBottom: 16 }}>
        <Form layout="vertical" onSubmit={handleGenerate} getFormApi={setGenFormApi}>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Select field="type" label="内容类型" initValue="article" style={{ width: '100%' }}>
                <Form.Select.Option value="article">文章 (Article)</Form.Select.Option>
                <Form.Select.Option value="prompt">提示词 (Prompt)</Form.Select.Option>
              </Form.Select>
            </Col>
            <Col span={16}>
              <Form.Input
                field="keywords"
                label="关键词（多个用逗号分隔）"
                placeholder="如：AI video prompt, Sora generation"
                rules={[{ required: true }]}
              />
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={6}>
              <Form.Select field="language" label="语言" initValue="en" style={{ width: '100%' }}>
                <Form.Select.Option value="en">English</Form.Select.Option>
                <Form.Select.Option value="zh">中文</Form.Select.Option>
                <Form.Select.Option value="ja">日本語</Form.Select.Option>
              </Form.Select>
            </Col>
            <Col span={18}>
              <Form.CheckboxGroup field="options" label="自动化选项" initValue={['auto_seo', 'auto_geo', 'auto_translate']}>
                <Form.Checkbox value="auto_seo">自动生成 SEO 字段</Form.Checkbox>
                <Form.Checkbox value="auto_geo">自动生成 GEO 结构化内容</Form.Checkbox>
                <Form.Checkbox value="auto_translate">自动翻译 12 种语言</Form.Checkbox>
                <Form.Checkbox value="auto_publish">自动发布</Form.Checkbox>
              </Form.CheckboxGroup>
            </Col>
          </Row>
          <Button type="primary" htmlType="submit" icon={<IconBolt />} loading={loading} style={{ marginTop: 12 }}>
            开始生成
          </Button>
        </Form>
      </Card>

      {queue.length > 0 && (
        <Card title="生成队列">
          {queue.map((task) => (
            <Card key={task.id} size="small" style={{ marginBottom: 8 }}>
              <Row justify="space-between" align="middle">
                <Col>
                  <Space>
                    {getStatusBadge(task.status)}
                    <Text strong>{task.keywords.join(', ')}</Text>
                    <Tag size="small">{task.type === 'article' ? '文章' : '提示词'}</Tag>
                  </Space>
                </Col>
                <Col>
                  <Space>
                    <Text type="secondary">{task.message}</Text>
                    {task.status === 'completed' && task.recordId && (
                      <Button
                        type="tertiary"
                        size="small"
                        onClick={() => {
                          if (task.type === 'article') {
                            navigate(`/console/article/editor/${task.recordId}`);
                          } else {
                            navigate('/console/prompt');
                          }
                        }}
                      >
                        查看
                      </Button>
                    )}
                  </Space>
                </Col>
              </Row>
              {task.status === 'generating' && (
                <Progress percent={task.progress} showInfo={false} style={{ marginTop: 8 }} />
              )}
            </Card>
          ))}
        </Card>
      )}
    </div>
  );
};

// ==================== Tab 3: Monitor ====================

const MonitorTab = () => {
  const { t } = useTranslation();
  const [data, setData] = useState(null);
  const [summary, setSummary] = useState(null);
  const [loading, setLoading] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [monitorRes, summaryRes] = await Promise.all([
        API.get('/api/admin/seo/monitor'),
        API.get('/api/admin/seo/monitor/summary'),
      ]);
      if (monitorRes.data.success) {
        setData(monitorRes.data.data);
      }
      if (summaryRes.data.success) {
        setSummary(summaryRes.data.data);
      }
    } catch (e) {
      console.error('load monitor data failed', e);
    }
    setLoading(false);
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const simulateData = async () => {
    try {
      const res = await API.post('/api/admin/seo/monitor/simulate');
      if (res.data.success) {
        setData(res.data.data);
        loadData();
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const getPositionColor = (pos) => {
    if (pos <= 3) return 'green';
    if (pos <= 10) return 'orange';
    return 'grey';
  };

  return (
    <div>
      <Row justify="end" style={{ marginBottom: 16 }}>
        <Button onClick={simulateData} size="small">模拟数据</Button>
        <Button onClick={loadData} icon={<IconRefresh />} size="small" style={{ marginLeft: 8 }}>刷新</Button>
      </Row>

      {loading && !data && (
        <Card style={{ textAlign: 'center', padding: 40 }}>
          <Spin size="large" />
        </Card>
      )}

      {data && (
        <>
          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={6}>
              <Card>
                <StatCard
                  title="自然搜索流量"
                  value={data.organic_traffic || 0}
                  suffix={summary?.traffic_change > 0 ? <Tag color="green">+{summary.traffic_change.toFixed(1)}%</Tag> : summary?.traffic_change < 0 ? <Tag color="red">{summary.traffic_change.toFixed(1)}%</Tag> : null}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <StatCard title="已索引页面" value={data.indexed_pages || 0} />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <StatCard title="排名关键词" value={data.ranking_keywords || 0} />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <StatCard title="平均排名" value={data.avg_position || 0} precision={1} />
              </Card>
            </Col>
          </Row>

          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={16}>
              <Card title="Top 排名关键词">
                <Table
                  columns={[
                    { title: '关键词', dataIndex: 'keyword' },
                    { title: '排名', dataIndex: 'position', render: (v) => <Tag color={getPositionColor(v)}>#{v}</Tag>, width: 80 },
                    { title: '点击', dataIndex: 'clicks', width: 80 },
                    { title: '展现', dataIndex: 'impressions', width: 100 },
                    { title: 'CTR', dataIndex: 'ctr', render: (v) => `${v.toFixed(1)}%`, width: 80 },
                    { title: '变化', dataIndex: 'change', render: (v) => v > 0 ? <Tag color="green">+{v}</Tag> : v < 0 ? <Tag color="red">{v}</Tag> : <Tag>-</Tag>, width: 80 },
                  ]}
                  dataSource={data.top_keywords || []}
                  pagination={false}
                  size="small"
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card title={`SEO 健康评分: ${data.health_score || 0}`}>
                <Progress percent={data.health_score || 0} size="large" />
                {data.issues?.length > 0 ? (
                  <div style={{ marginTop: 12 }}>
                    {data.issues.map((issue, idx) => (
                      <div key={idx} style={{ marginBottom: 8, display: 'flex', alignItems: 'center' }}>
                        {issue.type === 'error' && <IconAlertTriangle style={{ color: 'red', marginRight: 8 }} />}
                        {issue.type === 'warning' && <IconAlertTriangle style={{ color: 'orange', marginRight: 8 }} />}
                        {issue.type === 'info' && <IconInfoCircle style={{ color: 'blue', marginRight: 8 }} />}
                        <Text size="small">{issue.message} ({issue.count})</Text>
                        {issue.auto_fixable && <Tag size="small" style={{ marginLeft: 8 }}>可自动修复</Tag>}
                      </div>
                    ))}
                  </div>
                ) : (
                  <Empty description="暂无问题" />
                )}
              </Card>
            </Col>
          </Row>
        </>
      )}

      {!data && !loading && (
        <Card style={{ textAlign: 'center', padding: 40 }}>
          <Empty description="暂无监控数据，点击「模拟数据」查看效果" />
          <Button type="primary" onClick={simulateData} style={{ marginTop: 16 }}>模拟数据</Button>
        </Card>
      )}
    </div>
  );
};

// ==================== Main Page ====================

const SEOCenter = () => {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('research');
  const [generatorKeywords, setGeneratorKeywords] = useState('');

  const handleAddKeyword = (keyword) => {
    setGeneratorKeywords((prev) => {
      const existing = prev ? prev.split(/[,，]/).map((k) => k.trim()).filter(Boolean) : [];
      if (existing.includes(keyword)) {
        return prev;
      }
      return prev ? `${prev}, ${keyword}` : keyword;
    });
    setActiveTab('generator');
  };

  return (
    <div style={{ padding: 20 }}>
      <Title heading={3} style={{ marginBottom: 20 }}>SEO Center</Title>
      <Tabs activeKey={activeTab} onChange={setActiveTab} type="line">
        <TabPane tab={<span><IconSearch style={{ marginRight: 4 }} />关键词研究</span>} itemKey="research">
          <KeywordResearchTab onAddKeyword={handleAddKeyword} />
        </TabPane>
        <TabPane tab={<span><IconBolt style={{ marginRight: 4 }} />内容生成</span>} itemKey="generator">
          <ContentGeneratorTab generatorKeywords={generatorKeywords} />
        </TabPane>
        <TabPane tab={<span><IconEyeOpened style={{ marginRight: 4 }} />监控仪表盘</span>} itemKey="monitor">
          <MonitorTab />
        </TabPane>
      </Tabs>
    </div>
  );
};

export default SEOCenter;
