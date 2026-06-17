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
import React, { useEffect, useState } from 'react';
import {
  SideSheet,
  Spin,
  Card,
  Tag,
  Progress,
  Row,
  Col,
  Typography,
  List,
  Empty,
  Button,
  Space,
} from '@douyinfe/semi-ui';
import { IconAlertTriangle, IconTickCircle, IconInfoCircle } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError } from '../../helpers';

const { Title, Text } = Typography;

const CROSidebar = ({ recordId, contentType, visible, onClose }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState(null);

  useEffect(() => {
    if (!visible || !recordId) {
      setResult(null);
      return;
    }
    setLoading(true);
    API.post('/api/admin/cro/analyze', {
      record_id: recordId,
      content_type: contentType || 'article',
      language: 'zh',
    })
      .then((res) => {
        if (res.data.success) {
          setResult(res.data.data);
        } else {
          showError(res.data.message || t('CRO 分析失败'));
        }
      })
      .catch((err) => showError(err.message || t('CRO 分析请求失败')))
      .finally(() => setLoading(false));
  }, [visible, recordId, contentType]);

  const getStatusColor = (status) => {
    switch (status) {
      case 'pass': return 'green';
      case 'warning': return 'orange';
      case 'fail': return 'red';
      default: return 'grey';
    }
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'pass': return <IconTickCircle style={{ color: 'green' }} />;
      case 'warning': return <IconAlertTriangle style={{ color: 'orange' }} />;
      case 'fail': return <IconAlertTriangle style={{ color: 'red' }} />;
      default: return <IconInfoCircle style={{ color: 'grey' }} />;
    }
  };

  return (
    <SideSheet
      title={t('CRO 转化分析')}
      visible={visible}
      onCancel={onClose}
      width={520}
      bodyStyle={{ padding: 0 }}
      footer={
        <div className='flex justify-end'>
          <Button onClick={onClose}>{t('关闭')}</Button>
        </div>
      }
    >
      <Spin spinning={loading}>
        <div style={{ padding: 16 }}>
          {!recordId && (
            <Empty description={t('请先保存内容以获取 CRO 分析')} />
          )}

          {result && (
            <>
              <Card style={{ marginBottom: 16 }}>
                <Row gutter={16} align='middle'>
                  <Col span={16}>
                    <Title heading={6} style={{ margin: 0 }}>{result.title || t('未命名')}</Title>
                    <Text type='secondary' size='small'>
                      ID: {result.record_id} · {result.content_type === 'article' ? t('文章') : t('提示词')}
                    </Text>
                  </Col>
                  <Col span={8} style={{ textAlign: 'right' }}>
                    <Tag color={result.overall_score >= 70 ? 'green' : result.overall_score >= 50 ? 'orange' : 'red'}>
                      {result.overall_score} / 100
                    </Tag>
                  </Col>
                </Row>
                <Progress percent={result.overall_score} showInfo={false} style={{ marginTop: 12 }} />
              </Card>

              <Card title={t('维度评分')} style={{ marginBottom: 16 }}>
                <List
                  dataSource={result.dimensions || []}
                  renderItem={(dim) => (
                    <List.Item
                      main={
                        <div>
                          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 }}>
                            <Space>
                              {getStatusIcon(dim.status)}
                              <Text strong>{dim.name}</Text>
                            </Space>
                            <Tag color={getStatusColor(dim.status)}>{dim.score}</Tag>
                          </div>
                          <Progress percent={dim.score} showInfo={false} stroke={getStatusColor(dim.status)} />
                          <Text type='secondary' size='small'>{dim.suggestion}</Text>
                        </div>
                      }
                      style={{ padding: '12px 0' }}
                    />
                  )}
                />
              </Card>

              {result.ctas?.length > 0 && (
                <Card title={t('检测到的 CTA')} style={{ marginBottom: 16 }}>
                  <List
                    dataSource={result.ctas}
                    renderItem={(cta) => (
                      <List.Item
                        main={
                          <div>
                            <Text strong>“{cta.text}”</Text>
                            <div>
                              <Text type='secondary' size='small'>
                                {t('位置')}: {cta.position === 'top' ? t('顶部') : cta.position === 'middle' ? t('中部') : t('底部')}
                                {' · '}
                                {t('强度')}: {cta.strength}/10
                              </Text>
                            </div>
                            {cta.context && (
                              <Text type='secondary' size='small' style={{ display: 'block', marginTop: 4 }}>
                                {cta.context}
                              </Text>
                            )}
                          </div>
                        }
                      />
                    )}
                  />
                </Card>
              )}

              {result.ab_tests?.length > 0 && (
                <Card title={t('A/B 测试建议')} style={{ marginBottom: 16 }}>
                  <List
                    dataSource={result.ab_tests}
                    renderItem={(item, idx) => (
                      <List.Item main={<Text>{idx + 1}. {item}</Text>} />
                    )}
                  />
                </Card>
              )}

              {result.suggestions?.length > 0 && (
                <Card title={t('优化建议')}>
                  <List
                    dataSource={result.suggestions}
                    renderItem={(item, idx) => (
                      <List.Item main={<Text>{idx + 1}. {item}</Text>} />
                    )}
                  />
                </Card>
              )}
            </>
          )}
        </div>
      </Spin>
    </SideSheet>
  );
};

export default CROSidebar;
