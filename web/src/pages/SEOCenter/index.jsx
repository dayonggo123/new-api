/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
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
  Modal,
  Pagination,
  List,
  Banner,
} from '@douyinfe/semi-ui';
import {
  IconSearch,
  IconBolt,
  IconRefresh,
  IconDownload,
  IconEyeOpened,
  IconLanguage,
  IconTickCircle,
  IconAlertTriangle,
  IconInfoCircle,
  IconHistory,
  IconPlus,
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

const KeywordResearchTab = ({ onAddKeyword, onAddAllKeywords }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState(null);
  const [templates, setTemplates] = useState([]);
  const [formApi, setFormApi] = useState(null);
  const [historyVisible, setHistoryVisible] = useState(false);
  const [histories, setHistories] = useState([]);
  const [historyPage, setHistoryPage] = useState(1);
  const [historyTotal, setHistoryTotal] = useState(0);
  const [researchMode, setResearchMode] = useState('ai');
  const historyPageSize = 10;

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

  const loadHistoryList = async (page = 1) => {
    try {
      const res = await API.get(`/api/admin/seo/research/history?page=${page}&page_size=${historyPageSize}`);
      if (res.data.success) {
        setHistories(res.data.data.histories || []);
        setHistoryTotal(res.data.data.total || 0);
        setHistoryPage(res.data.data.page || 1);
      }
    } catch (e) {
      console.error('load history failed', e);
    }
  };

  const loadHistoryDetail = async (id) => {
    try {
      const res = await API.get(`/api/admin/seo/research/history/${id}`);
      if (res.data.success && res.data.data) {
        const history = res.data.data;
        if (history.result_json) {
          setResult(JSON.parse(history.result_json));
          setHistoryVisible(false);
          showSuccess('已加载历史研究记录');
        }
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const deleteHistory = async (id) => {
    try {
      const res = await API.delete(`/api/admin/seo/research/history/${id}`);
      if (res.data.success) {
        showSuccess('已删除');
        loadHistoryList(historyPage);
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const handleResearch = async (values) => {
    if (!values.seed_keyword) {
      showError('请输入研究主题');
      return;
    }
    const mode = values.research_mode || researchMode;
    setLoading(true);
    try {
      const endpointMap = {
        ai: '/api/admin/seo/research',
        serp: '/api/admin/seo/research/serp',
        site: '/api/admin/seo/research/site',
        competitor: '/api/admin/seo/research/competitor',
        community: '/api/admin/seo/research/community',
      };
      const endpoint = endpointMap[mode] || '/api/admin/seo/research';
      const res = await API.post(endpoint, {
        seed_keyword: values.seed_keyword,
        language: values.language || 'en',
      });
      if (res.data.success) {
        setResult(res.data.data);
        const successMap = {
          ai: '关键词研究完成',
          serp: 'SERP 深挖完成',
          site: '站内机会词分析完成',
          competitor: '竞品反查完成',
          community: '社群挖掘完成',
        };
        showSuccess(successMap[mode] || '研究完成');
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

  const collectAllKeywords = () => {
    const all = new Set();
    (result.high_roi_keywords || []).forEach((item) => all.add(item.keyword));
    (result.seed_keywords || []).forEach((item) => all.add(item.keyword));
    (result.extended_keywords || []).forEach((item) => all.add(item.keyword));
    (result.content_gaps || []).forEach((item) => all.add(item.keyword));
    (result.topic_clusters || []).forEach((cluster) => {
      all.add(cluster.pillar_keyword);
      (cluster.cluster_keywords || []).forEach((kw) => all.add(kw));
    });
    return Array.from(all);
  };

  const handleAddAll = () => {
    if (!onAddAllKeywords) return;
    const allKeywords = collectAllKeywords();
    if (allKeywords.length === 0) {
      showError('没有可添加的关键词');
      return;
    }
    onAddAllKeywords(allKeywords);
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
            <Col span={10}>
              <Form.Input
                field="seed_keyword"
                label="研究主题"
                placeholder="输入关键词主题，如：AI video prompt library"
                rules={[{ required: true }]}
              />
            </Col>
            <Col span={5}>
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
            <Col span={5}>
                <Form.Select
                field="research_mode"
                label="研究模式"
                initValue="ai"
                style={{ width: '100%' }}
                onChange={(value) => setResearchMode(value)}
              >
                <Form.Select.Option value="ai">AI 主题探索</Form.Select.Option>
                <Form.Select.Option value="serp">SERP 深挖</Form.Select.Option>
                <Form.Select.Option value="site">站内机会词</Form.Select.Option>
                <Form.Select.Option value="competitor">竞品反查</Form.Select.Option>
                <Form.Select.Option value="community">社群挖掘</Form.Select.Option>
              </Form.Select>
            </Col>
            <Col span={4}>
              <Space style={{ marginTop: 28 }}>
                <Button type="primary" htmlType="submit" icon={<IconSearch />} loading={loading}>
                  开始研究
                </Button>
                <Button
                  type="tertiary"
                  icon={<IconHistory />}
                  onClick={() => {
                    setHistoryVisible(true);
                    loadHistoryList(1);
                  }}
                >
                  历史记录
                </Button>
              </Space>
            </Col>
          </Row>
        </Form>

        {templates.length > 0 && (
          <div style={{ marginTop: 16 }}>
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
              <Text type="secondary" style={{ marginRight: 8, flexShrink: 0 }}>快速模板：</Text>
              <div
                style={{
                  display: 'flex',
                  gap: 8,
                  overflowX: 'auto',
                  paddingBottom: 4,
                  scrollbarWidth: 'thin',
                }}
              >
                {templates.map((tmpl) => (
                  <Tag
                    key={tmpl.id}
                    type="light"
                    style={{
                      cursor: 'pointer',
                      whiteSpace: 'nowrap',
                      flexShrink: 0,
                      padding: '4px 12px',
                      fontSize: 13,
                    }}
                    onClick={() => {
                      if (formApi) {
                        formApi.setValue('seed_keyword', tmpl.seed_keyword);
                        const mode = tmpl.research_mode || 'ai';
                        formApi.setValue('research_mode', mode);
                        setResearchMode(mode);
                      }
                    }}
                    title={tmpl.description}
                  >
                    {tmpl.name}
                  </Tag>
                ))}
              </div>
            </div>
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
                <div style={{ textAlign: 'center', marginTop: 8 }}>
                  <Button
                    type="primary"
                    size="small"
                    icon={<IconPlus />}
                    onClick={handleAddAll}
                  >
                    全部添加到生成
                  </Button>
                </div>
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

          {result.long_tail_keywords?.length > 0 && (
            <Card title="真实搜索建议（来自 Google Suggest）" style={{ marginBottom: 16 }}>
              <Table
                columns={columns}
                dataSource={result.long_tail_keywords || []}
                pagination={{ pageSize: 10 }}
                size="small"
              />
            </Card>
          )}

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
                {
                  title: '缺口类型',
                  dataIndex: 'gap_type',
                  render: (v) => {
                    const map = {
                      faq_opportunity: 'FAQ 机会',
                      missing_topic: '缺失主题',
                      undercovered: '覆盖不足',
                      poor_quality: '质量差',
                    };
                    return <Tag size="small">{map[v] || v}</Tag>;
                  },
                  width: 90,
                },
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

      <Modal
        title="研究历史记录"
        visible={historyVisible}
        onCancel={() => setHistoryVisible(false)}
        footer={null}
        width={720}
      >
        <List
          dataSource={histories}
          renderItem={(item) => (
            <List.Item
              main={
                <div>
                  <Text strong>{item.seed_keyword}</Text>
                  <div>
                    <Text type="secondary" size="small">
                      {item.language} · {item.total_count} 个关键词 · {new Date(item.created_time * 1000).toLocaleString()}
                    </Text>
                  </div>
                </div>
              }
              extra={
                <Space>
                  <Button type="primary" size="small" onClick={() => loadHistoryDetail(item.id)}>
                    加载
                  </Button>
                  <Button type="danger" size="small" onClick={() => deleteHistory(item.id)}>
                    删除
                  </Button>
                </Space>
              }
            />
          )}
        />
        {historyTotal > historyPageSize && (
          <div style={{ textAlign: 'center', marginTop: 16 }}>
            <Pagination
              currentPage={historyPage}
              pageSize={historyPageSize}
              total={historyTotal}
              onChange={(page) => loadHistoryList(page)}
            />
          </div>
        )}
      </Modal>
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

  // 当从关键词研究页面添加关键词时，自动填充到输入框（自动去重，只追加新增关键词）
  useEffect(() => {
    if (genFormApi && generatorKeywords) {
      const current = genFormApi.getValue('keywords') || '';
      const currentList = current.split(/[,，]/).map((k) => k.trim()).filter(Boolean);
      const newList = generatorKeywords.split(/[,，]/).map((k) => k.trim()).filter(Boolean);
      const toAdd = newList.filter((k) => !currentList.includes(k));
      if (toAdd.length > 0) {
        const newValue = current ? `${current}, ${toAdd.join(', ')}` : toAdd.join(', ');
        genFormApi.setValue('keywords', newValue);
      }
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
        intent: values.intent || 'informational',
        word_count: values.word_count || (values.type === 'tutorial' ? 2000 : 2500),
        cta_goal: values.cta_goal || '',
        auto_seo: auto_seo,
        auto_geo: auto_geo,
        auto_translate: auto_translate,
        auto_publish: auto_publish,
      });

      if (res.data.success) {
        const data = res.data.data;
        const taskStatus = data.status === 'completed_with_warning' ? 'warning' : 'completed';
        const taskMessage = data.status === 'completed_with_warning'
          ? `已生成但门禁未通过：${data.gate_message || '评分不足'}（ID: ${data.record_id}）`
          : `完成！ID: ${data.record_id}`;
        setQueue((prev) =>
          prev.map((t) =>
            t.id === task.id
              ? { ...t, status: taskStatus, progress: 100, message: taskMessage, recordId: data.record_id }
              : t
          )
        );
        if (data.status === 'completed_with_warning') {
          showError(data.gate_message || '内容已生成，但发布前质量门禁未通过，已保存为草稿');
        } else {
          showSuccess('内容生成完成');
        }
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
      case 'warning': return <Badge type="warning" text="门禁未通过" />;
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
                <Form.Select.Option value="tutorial">教程 (Tutorial)</Form.Select.Option>
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
            <Col span={6}>
              <Form.Select field="intent" label="搜索意图" initValue="informational" style={{ width: '100%' }}>
                <Form.Select.Option value="informational">信息型</Form.Select.Option>
                <Form.Select.Option value="commercial">商业调查型</Form.Select.Option>
                <Form.Select.Option value="transactional">交易型</Form.Select.Option>
                <Form.Select.Option value="navigational">导航型</Form.Select.Option>
              </Form.Select>
            </Col>
            <Col span={6}>
              <Form.InputNumber
                field="word_count"
                label="目标字数"
                initValue={2500}
                min={500}
                max={10000}
                step={100}
                style={{ width: '100%' }}
              />
            </Col>
            <Col span={6}>
              <Form.Input
                field="cta_goal"
                label="CTA 目标"
                placeholder="如：引导用户注册试用、收藏提示词、阅读教程"
              />
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={24}>
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
                    {(task.status === 'completed' || task.status === 'warning') && task.recordId && (
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
  const [lowCtrLoading, setLowCtrLoading] = useState(false);
  const [lowCtrItems, setLowCtrItems] = useState([]);
  const [rankingDropLoading, setRankingDropLoading] = useState(false);
  const [rankingDropItems, setRankingDropItems] = useState([]);
  const [queueVisible, setQueueVisible] = useState(false);
  const [queue, setQueue] = useState([]);
  const [queueLoading, setQueueLoading] = useState(false);
  const [syncGSCLoading, setSyncGSCLoading] = useState(false);

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

  const loadLowCtr = useCallback(async () => {
    setLowCtrLoading(true);
    try {
      const res = await API.get('/api/admin/seo/optimization-queue/low-ctr?ctr_threshold=5.0&min_impressions=100');
      if (res.data.success) {
        setLowCtrItems(res.data.data?.items || []);
      }
    } catch (e) {
      console.error('load low ctr failed', e);
    }
    setLowCtrLoading(false);
  }, []);

  const loadRankingDrop = useCallback(async () => {
    setRankingDropLoading(true);
    try {
      const res = await API.get('/api/admin/seo/optimization-queue/ranking-drop?position_threshold=10.0&change_threshold=-1.0');
      if (res.data.success) {
        setRankingDropItems(res.data.data?.items || []);
      }
    } catch (e) {
      console.error('load ranking drop failed', e);
    }
    setRankingDropLoading(false);
  }, []);

  const loadQueue = useCallback(async () => {
    setQueueLoading(true);
    try {
      const res = await API.get('/api/admin/seo/optimization-queue?status=all&limit=50');
      if (res.data.success) {
        setQueue(res.data.data?.items || []);
      }
    } catch (e) {
      console.error('load queue failed', e);
    }
    setQueueLoading(false);
  }, []);

  useEffect(() => {
    loadData();
    loadLowCtr();
    loadRankingDrop();
  }, [loadData, loadLowCtr, loadRankingDrop]);

  const addToQueue = async (item, reason = 'low_ctr') => {
    try {
      const res = await API.post('/api/admin/seo/optimization-queue', {
        record_id: 0,
        content_type: 'keyword',
        title: item.keyword,
        keyword: item.keyword,
        reason: reason,
        score_before: Math.round(item.ctr * 10),
        extra: JSON.stringify({ position: item.position, impressions: item.impressions, clicks: item.clicks, change: item.change, reason: item.reason }),
      });
      if (res.data.success) {
        showSuccess(`已把“${item.keyword}”加入优化队列`);
        loadQueue();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const updateQueueStatus = async (id, status) => {
    try {
      const res = await API.put(`/api/admin/seo/optimization-queue/${id}/status`, { status });
      if (res.data.success) {
        showSuccess('状态已更新');
        loadQueue();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e.message);
    }
  };

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

  const syncFromGSC = async () => {
    setSyncGSCLoading(true);
    try {
      const res = await API.post('/api/admin/seo/monitor/sync-gsc', {});
      if (res.data.success) {
        showSuccess('已从 Google Search Console 同步数据');
        setData(res.data.data?.data);
        loadLowCtr();
        loadRankingDrop();
      } else {
        showError(res.data.message || '同步失败');
      }
    } catch (e) {
      showError(e.message || '同步失败，请检查 GSC 配置');
    }
    setSyncGSCLoading(false);
  };

  const getPositionColor = (pos) => {
    if (pos <= 3) return 'green';
    if (pos <= 10) return 'orange';
    return 'grey';
  };

  return (
    <div>
      {data?.is_simulated && (
        <Banner
          type="warning"
          title="当前展示的是模拟数据"
          description="点击下方「模拟数据」按钮生成、或调用 POST /api/admin/seo/monitor/update 接入真实数据源（如 Google Search Console）。请勿将模拟数据用于运营决策。"
          closeable
        />
      )}
      <Row justify="end" style={{ marginBottom: 16 }}>
        <Button onClick={simulateData} size="small">模拟数据</Button>
        <Button onClick={loadData} icon={<IconRefresh />} size="small" style={{ marginLeft: 8 }}>刷新</Button>
        <Button type="primary" onClick={syncFromGSC} icon={<IconDownload />} loading={syncGSCLoading} size="small" style={{ marginLeft: 8 }}>从 GSC 同步</Button>
        <Button onClick={() => { setQueueVisible(true); loadQueue(); }} size="small" style={{ marginLeft: 8 }}>优化队列</Button>
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

      {lowCtrItems.length > 0 && (
        <Card title={`低 CTR 回流机会（${lowCtrItems.length} 个）`} style={{ marginBottom: 16 }} loading={lowCtrLoading}>
          <Table
            columns={[
              { title: '关键词', dataIndex: 'keyword' },
              { title: '排名', dataIndex: 'position', render: (v) => <Tag color={v <= 3 ? 'green' : v <= 10 ? 'orange' : 'grey'}>#{v}</Tag>, width: 80 },
              { title: '展现', dataIndex: 'impressions', width: 100 },
              { title: '点击', dataIndex: 'clicks', width: 80 },
              { title: 'CTR', dataIndex: 'ctr', render: (v) => <Tag color={v < 3 ? 'red' : 'orange'}>{v.toFixed(1)}%</Tag>, width: 90 },
              {
                title: '操作',
                key: 'action',
                width: 140,
                render: (_, record) => (
                  <Button type='primary' size='small' icon={<IconPlus />} onClick={() => addToQueue(record, 'low_ctr')}>
                    加入队列
                  </Button>
                ),
              },
            ]}
            dataSource={lowCtrItems}
            pagination={{ pageSize: 5 }}
            size='small'
          />
        </Card>
      )}

      {rankingDropItems.length > 0 && (
        <Card title={`排名下降 / 低位回流机会（${rankingDropItems.length} 个）`} style={{ marginBottom: 16 }} loading={rankingDropLoading}>
          <Table
            columns={[
              { title: '关键词', dataIndex: 'keyword' },
              { title: '排名', dataIndex: 'position', render: (v) => <Tag color={v <= 3 ? 'green' : v <= 10 ? 'orange' : 'red'}>#{v}</Tag>, width: 80 },
              { title: '变化', dataIndex: 'change', render: (v) => v > 0 ? <Tag color="green">+{v}</Tag> : v < 0 ? <Tag color="red">{v}</Tag> : <Tag>-</Tag>, width: 80 },
              { title: '展现', dataIndex: 'impressions', width: 100 },
              { title: '点击', dataIndex: 'clicks', width: 80 },
              { title: 'CTR', dataIndex: 'ctr', render: (v) => `${v.toFixed(1)}%`, width: 90 },
              { title: '原因', dataIndex: 'reason', render: (v) => <Tag size='small'>{v}</Tag>, width: 140 },
              {
                title: '操作',
                key: 'action',
                width: 140,
                render: (_, record) => (
                  <Button type='primary' size='small' icon={<IconPlus />} onClick={() => addToQueue(record, 'ranking_drop')}>
                    加入队列
                  </Button>
                ),
              },
            ]}
            dataSource={rankingDropItems}
            pagination={{ pageSize: 5 }}
            size='small'
          />
        </Card>
      )}

      <Modal
        title='SEO 优化队列'
        visible={queueVisible}
        onCancel={() => setQueueVisible(false)}
        footer={null}
        width={800}
      >
        <Spin spinning={queueLoading}>
          <Table
            columns={[
              { title: 'ID', dataIndex: 'id', width: 60 },
              { title: '标题/关键词', dataIndex: 'title' },
              { title: '类型', dataIndex: 'content_type', width: 90 },
              { title: '原因', dataIndex: 'reason', render: (v) => <Tag size='small'>{v}</Tag>, width: 100 },
              { title: '优化前分数', dataIndex: 'score_before', width: 100 },
              {
                title: '状态',
                dataIndex: 'status',
                width: 100,
                render: (v) => {
                  const color = v === 'optimized' ? 'green' : v === 'processing' ? 'blue' : v === 'dismissed' ? 'grey' : 'orange';
                  const label = { pending: '待处理', processing: '处理中', optimized: '已优化', dismissed: '已忽略' }[v] || v;
                  return <Tag color={color}>{label}</Tag>;
                },
              },
              {
                title: '操作',
                key: 'action',
                width: 180,
                render: (_, record) => (
                  <Space>
                    <Button type='tertiary' size='small' onClick={() => updateQueueStatus(record.id, 'processing')}>处理中</Button>
                    <Button type='tertiary' size='small' onClick={() => updateQueueStatus(record.id, 'optimized')}>完成</Button>
                    <Button type='tertiary' size='small' onClick={() => updateQueueStatus(record.id, 'dismissed')}>忽略</Button>
                  </Space>
                ),
              },
            ]}
            dataSource={queue}
            pagination={{ pageSize: 10 }}
            size='small'
          />
        </Spin>
      </Modal>

      {!data && !loading && (
        <Card style={{ textAlign: 'center', padding: 40 }}>
          <Empty description="暂无监控数据，点击「模拟数据」查看效果" />
          <Button type="primary" onClick={simulateData} style={{ marginTop: 16 }}>模拟数据</Button>
        </Card>
      )}
    </div>
  );
};

// ==================== Tab 4: Content Optimize ====================

const ContentOptimizeTab = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState(null);

  const handleOptimize = async (values) => {
    if (!values.record_id) {
      showError('请输入内容 ID');
      return;
    }
    setLoading(true);
    try {
      const res = await API.post('/api/admin/content/optimize', {
        record_id: parseInt(values.record_id, 10),
        content_type: values.content_type || 'article',
        language: values.language || 'en',
      });
      if (res.data.success) {
        setResult(res.data.data);
        showSuccess('内容优化联审完成');
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e.message);
    }
    setLoading(false);
  };

  const getScoreColor = (score) => {
    if (score >= 80) return 'green';
    if (score >= 70) return 'orange';
    return 'red';
  };

  const getStatusBadge = (status) => {
    switch (status) {
      case 'ready': return <Badge type="success" text="可发布" />;
      case 'needs_fix': return <Badge type="warning" text="需修复" />;
      case 'critical': return <Badge type="danger" text="不建议发布" />;
      default: return <Badge type="default" text={status} />;
    }
  };

  return (
    <div>
      <Card style={{ marginBottom: 16 }}>
        <Form layout="horizontal" onSubmit={handleOptimize}>
          <Row gutter={16} align="middle">
            <Col span={6}>
              <Form.Select field="content_type" label="内容类型" initValue="article" style={{ width: '100%' }}>
                <Form.Select.Option value="article">文章 (Article)</Form.Select.Option>
                <Form.Select.Option value="prompt">提示词 (Prompt)</Form.Select.Option>
              </Form.Select>
            </Col>
            <Col span={6}>
              <Form.Input
                field="record_id"
                label="内容 ID"
                placeholder="如：123"
                rules={[{ required: true }]}
              />
            </Col>
            <Col span={6}>
              <Form.Select field="language" label="语言" initValue="en" style={{ width: '100%' }}>
                <Form.Select.Option value="en">English</Form.Select.Option>
                <Form.Select.Option value="zh">中文</Form.Select.Option>
                <Form.Select.Option value="ja">日本語</Form.Select.Option>
                <Form.Select.Option value="ko">한국어</Form.Select.Option>
                <Form.Select.Option value="es">Español</Form.Select.Option>
                <Form.Select.Option value="de">Deutsch</Form.Select.Option>
                <Form.Select.Option value="fr">Français</Form.Select.Option>
              </Form.Select>
            </Col>
            <Col span={6}>
              <Button type="primary" htmlType="submit" icon={<IconTickCircle />} loading={loading} style={{ marginTop: 28 }}>
                开始优化联审
              </Button>
            </Col>
          </Row>
        </Form>
      </Card>

      {loading && (
        <Card style={{ textAlign: 'center', padding: 40 }}>
          <Spin size="large" tip={<span style={{ whiteSpace: 'nowrap' }}>正在进行 SEO / 可读性 / 内链综合评估...</span>} />
        </Card>
      )}

      {result && !loading && (
        <>
          <Card style={{ marginBottom: 16 }}>
            <Row gutter={16} align="middle">
              <Col span={16}>
                <Title heading={5} style={{ margin: 0 }}>{result.title || '未命名内容'}</Title>
                <Text type="secondary">ID: {result.record_id} · 类型: {result.content_type === 'article' ? '文章' : '提示词'}</Text>
              </Col>
              <Col span={8} style={{ textAlign: 'right' }}>
                {getStatusBadge(result.status)}
              </Col>
            </Row>
          </Card>

          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={8}>
              <Card>
                <StatCard title="SEO 评分" value={result.seo_score} />
                <div style={{ textAlign: 'center', marginTop: 8 }}>
                  <Tag color={getScoreColor(result.seo_score)}>{result.seo_score >= 70 ? '达标' : '未达标'}</Tag>
                </div>
              </Card>
            </Col>
            <Col span={8}>
              <Card>
                <StatCard title="人性化评分" value={result.human_score} />
                <div style={{ textAlign: 'center', marginTop: 8 }}>
                  <Tag color={getScoreColor(result.human_score)}>{result.human_score >= 70 ? '达标' : '未达标'}</Tag>
                </div>
              </Card>
            </Col>
            <Col span={8}>
              <Card>
                <StatCard title="可读性评分" value={result.readability_score} />
                <div style={{ textAlign: 'center', marginTop: 8 }}>
                  <Tag color={getScoreColor(result.readability_score)}>{result.readability_score >= 70 ? '达标' : '未达标'}</Tag>
                </div>
              </Card>
            </Col>
          </Row>

          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={12}>
              <Card title="维度评分">
                {(result.dimension_scores || []).map((dim, idx) => (
                  <div key={idx} style={{ marginBottom: 12 }}>
                    <Row justify="space-between" align="middle">
                      <Col>
                        <Text strong>{dim.name}</Text>
                        <div><Text type="secondary" size="small">{dim.description}</Text></div>
                      </Col>
                      <Col>
                        <Tag color={getScoreColor(dim.score)}>{dim.score}</Tag>
                      </Col>
                    </Row>
                    <Progress percent={dim.score} showInfo={false} stroke={getScoreColor(dim.score)} style={{ marginTop: 4 }} />
                  </div>
                ))}
              </Card>
            </Col>
            <Col span={12}>
              <Card title="发布前检查清单">
                <List
                  dataSource={result.publish_check || []}
                  renderItem={(item) => (
                    <List.Item
                      main={
                        <div style={{ display: 'flex', alignItems: 'center' }}>
                          {item.passed ? (
                            <IconTickCircle style={{ color: 'green', marginRight: 8 }} />
                          ) : (
                            <IconAlertTriangle style={{ color: 'red', marginRight: 8 }} />
                          )}
                          <div>
                            <Text>{item.item}</Text>
                            <div><Text type="secondary" size="small">{item.message}</Text></div>
                          </div>
                        </div>
                      }
                    />
                  )}
                />
              </Card>
            </Col>
          </Row>

          {result.suggestions?.length > 0 && (
            <Card title="优化建议" style={{ marginBottom: 16 }}>
              <List
                dataSource={result.suggestions}
                renderItem={(item, idx) => (
                  <List.Item main={<Text>{idx + 1}. {item}</Text>} />
                )}
              />
            </Card>
          )}

          {result.issues?.length > 0 && (
            <Card title="SEO 问题" style={{ marginBottom: 16 }}>
              <List
                dataSource={result.issues}
                renderItem={(issue) => (
                  <List.Item
                    main={
                      <div>
                        <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
                          <Tag color={issue.type === 'error' ? 'red' : issue.type === 'warning' ? 'orange' : 'grey'} size="small">
                            {issue.type}
                          </Tag>
                          <Text strong style={{ marginLeft: 8 }}>{issue.field}</Text>
                        </div>
                        <Text>{issue.message}</Text>
                        <div><Text type="secondary" size="small">建议：{issue.suggestion}</Text></div>
                      </div>
                    }
                  />
                )}
              />
            </Card>
          )}

          {result.internal_links?.length > 0 && (
            <Card title="内链推荐" style={{ marginBottom: 16 }}>
              <Table
                columns={[
                  { title: '目标页面', dataIndex: 'target_title' },
                  { title: '锚文本', dataIndex: 'anchor_text' },
                  { title: '相关度', dataIndex: 'relevance', render: (v) => <Progress percent={v} size="small" showInfo={true} />, width: 120 },
                  { title: '原因', dataIndex: 'reason' },
                  {
                    title: '操作',
                    key: 'action',
                    width: 100,
                    render: (_, record) => (
                      <Button
                        type="tertiary"
                        size="small"
                        onClick={() => navigate(`/console/${record.target_type === 'article' ? 'article' : 'prompt'}`)}
                      >
                        查看
                      </Button>
                    ),
                  },
                ]}
                dataSource={result.internal_links}
                pagination={{ pageSize: 5 }}
                size="small"
              />
            </Card>
          )}
        </>
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

  const handleAddAllKeywords = (keywords) => {
    setGeneratorKeywords((prev) => {
      const existing = prev ? prev.split(/[,，]/).map((k) => k.trim()).filter(Boolean) : [];
      const toAdd = keywords.filter((k) => !existing.includes(k));
      if (toAdd.length === 0) return prev;
      return prev ? `${prev}, ${toAdd.join(', ')}` : toAdd.join(', ');
    });
    setActiveTab('generator');
  };

  return (
    <div style={{ padding: 20 }}>
      <Title heading={3} style={{ marginBottom: 20 }}>SEO Center</Title>
      <Tabs activeKey={activeTab} onChange={setActiveTab} type="line">
        <TabPane tab={<span><IconSearch style={{ marginRight: 4 }} />关键词研究</span>} itemKey="research">
          <KeywordResearchTab onAddKeyword={handleAddKeyword} onAddAllKeywords={handleAddAllKeywords} />
        </TabPane>
        <TabPane tab={<span><IconBolt style={{ marginRight: 4 }} />内容生成</span>} itemKey="generator">
          <ContentGeneratorTab generatorKeywords={generatorKeywords} />
        </TabPane>
        <TabPane tab={<span><IconEyeOpened style={{ marginRight: 4 }} />监控仪表盘</span>} itemKey="monitor">
          <MonitorTab />
        </TabPane>
        <TabPane tab={<span><IconTickCircle style={{ marginRight: 4 }} />内容优化</span>} itemKey="optimize">
          <ContentOptimizeTab />
        </TabPane>
      </Tabs>
    </div>
  );
};

export default SEOCenter;
