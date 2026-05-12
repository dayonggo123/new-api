import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Spin,
  Typography,
  Tag,
  Empty,
  Row,
  Col,
} from '@douyinfe/semi-ui';
import { VChart } from '@visactor/react-vchart';
import { initVChartSemiTheme } from '@visactor/vchart-semi-theme';
import { IconAlertTriangle, IconDownload } from '@douyinfe/semi-icons';
import { API, showError } from '../../helpers';

const CHART_OPTION = { mode: 'desktop-browser' };

const { Title, Text } = Typography;

const SEOTrends = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [trends, setTrends] = useState([]);
  const [lowScores, setLowScores] = useState([]);
  const [stats, setStats] = useState(null);

  useEffect(() => {
    initVChartSemiTheme({ isWatchingThemeSwitch: true });
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [trendsRes, lowRes, statsRes] = await Promise.all([
        API.get('/api/prompt/seo/trends?days=30'),
        API.get('/api/prompt/seo/low-score?threshold=60&limit=20'),
        API.get('/api/prompt/seo/stats'),
      ]);
      if (trendsRes.data.success) setTrends(trendsRes.data.data || []);
      if (lowRes.data.success) setLowScores(lowRes.data.data || []);
      if (statsRes.data.success) setStats(statsRes.data.data);
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  };

  const scoreSpec = {
    type: 'line',
    data: [{ id: 'scoreData', values: trends }],
    xField: 'date',
    yField: 'avg_score',
    line: {
      style: { curveType: 'monotone', lineWidth: 3 },
      point: { style: { size: 4 } },
    },
    axes: [
      { orient: 'bottom', label: { formatMethod: (v) => v.slice(5) } },
      { orient: 'left', max: 100, min: 0 },
    ],
    title: { visible: true, text: t('平均审计分趋势') },
    tooltip: {
      dimension: { content: [{ key: 'date', title: t('日期') }, { key: 'avg_score', title: t('平均分') }] },
    },
  };

  const countSpec = {
    type: 'line',
    data: [{ id: 'countData', values: trends }],
    xField: 'date',
    yField: 'audit_count',
    line: {
      style: { curveType: 'monotone', lineWidth: 3, stroke: '#10b981' },
      point: { style: { size: 4 } },
    },
    axes: [
      { orient: 'bottom', label: { formatMethod: (v) => v.slice(5) } },
      { orient: 'left' },
    ],
    title: { visible: true, text: t('每日审计数量趋势') },
    tooltip: {
      dimension: { content: [{ key: 'date', title: t('日期') }, { key: 'audit_count', title: t('审计数') }] },
    },
  };

  const coverageSpec = {
    type: 'line',
    data: [{ id: 'covData', values: trends }],
    xField: 'date',
    yField: 'prompt_count',
    line: {
      style: { curveType: 'monotone', lineWidth: 3, stroke: '#8b5cf6' },
      point: { style: { size: 4 } },
    },
    axes: [
      { orient: 'bottom', label: { formatMethod: (v) => v.slice(5) } },
      { orient: 'left' },
    ],
    title: { visible: true, text: t('每日覆盖 Prompt 数') },
    tooltip: {
      dimension: { content: [{ key: 'date', title: t('日期') }, { key: 'prompt_count', title: t('Prompt 数') }] },
    },
  };

  const handleExportReport = async () => {
    try {
      const res = await API.get('/api/prompt/seo/report-all', { responseType: 'blob' });
      const url = window.URL.createObjectURL(new Blob([res.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', 'seo-report-all.md');
      document.body.appendChild(link);
      link.click();
      link.remove();
    } catch (error) {
      showError(t('导出失败'));
    }
  };

  return (
    <div className='mt-[60px] px-2'>
      <Spin spinning={loading}>
        {/* 统计卡片 */}
        {stats && (
          <Row gutter={12} className='mb-4'>
            <Col span={6} xs={24} sm={12} md={6}>
              <Card className='!rounded-2xl shadow-sm border-0' bodyStyle={{ padding: '16px' }}>
                <div className='text-sm text-gray-500 mb-1'>{t('SEO 覆盖率')}</div>
                <div className='text-2xl font-bold' style={{ color: stats.seo_coverage >= 80 ? '#52c41a' : stats.seo_coverage >= 50 ? '#faad14' : '#f5222d' }}>
                  {stats.seo_coverage}%
                </div>
              </Card>
            </Col>
            <Col span={6} xs={24} sm={12} md={6}>
              <Card className='!rounded-2xl shadow-sm border-0' bodyStyle={{ padding: '16px' }}>
                <div className='text-sm text-gray-500 mb-1'>{t('审计覆盖率')}</div>
                <div className='text-2xl font-bold' style={{ color: stats.audit_coverage >= 80 ? '#52c41a' : stats.audit_coverage >= 50 ? '#faad14' : '#f5222d' }}>
                  {stats.audit_coverage}%
                </div>
              </Card>
            </Col>
            <Col span={6} xs={24} sm={12} md={6}>
              <Card className='!rounded-2xl shadow-sm border-0' bodyStyle={{ padding: '16px' }}>
                <div className='text-sm text-gray-500 mb-1'>{t('平均审计分')}</div>
                <div className='text-2xl font-bold' style={{ color: stats.average_score >= 80 ? '#52c41a' : stats.average_score >= 60 ? '#faad14' : '#f5222d' }}>
                  {stats.average_score}
                </div>
              </Card>
            </Col>
            <Col span={6} xs={24} sm={12} md={6}>
              <Card className='!rounded-2xl shadow-sm border-0' bodyStyle={{ padding: '16px' }}>
                <div className='text-sm text-gray-500 mb-1'>{t('低分数量')}</div>
                <div className='text-2xl font-bold' style={{ color: lowScores.length === 0 ? '#52c41a' : '#f5222d' }}>
                  {lowScores.length}
                </div>
              </Card>
            </Col>
          </Row>
        )}

        {/* 导出按钮 */}
        <div className='flex justify-end mb-4'>
          <Button icon={<IconDownload />} onClick={handleExportReport}>
            {t('导出全站 SEO 报告')}
          </Button>
        </div>

        {/* 趋势图表 */}
        <Row gutter={12} className='mb-4'>
          <Col span={24} md={12} className='mb-4 md:mb-0'>
            <Card className='!rounded-2xl shadow-sm border-0' bodyStyle={{ padding: '12px' }}>
              <div className='h-72'>
                {trends.length > 0 ? (
                  <VChart spec={scoreSpec} option={CHART_OPTION} />
                ) : (
                  <Empty description={t('暂无数据')} />
                )}
              </div>
            </Card>
          </Col>
          <Col span={24} md={12}>
            <Card className='!rounded-2xl shadow-sm border-0' bodyStyle={{ padding: '12px' }}>
              <div className='h-72'>
                {trends.length > 0 ? (
                  <VChart spec={countSpec} option={CHART_OPTION} />
                ) : (
                  <Empty description={t('暂无数据')} />
                )}
              </div>
            </Card>
          </Col>
        </Row>

        <Row gutter={12} className='mb-4'>
          <Col span={24}>
            <Card className='!rounded-2xl shadow-sm border-0' bodyStyle={{ padding: '12px' }}>
              <div className='h-72'>
                {trends.length > 0 ? (
                  <VChart spec={coverageSpec} option={CHART_OPTION} />
                ) : (
                  <Empty description={t('暂无数据')} />
                )}
              </div>
            </Card>
          </Col>
        </Row>

        {/* 低分 Prompt 列表 */}
        <Card className='!rounded-2xl shadow-sm border-0 mb-4' title={
          <div className='flex items-center gap-2'>
            <IconAlertTriangle size={16} style={{ color: '#f5222d' }} />
            <span>{t('低分提示词（需改进）')}</span>
          </div>
        }>
          {lowScores.length === 0 ? (
            <Empty description={t('暂无低分提示词')} />
          ) : (
            <div className='overflow-x-auto'>
              <table className='w-full text-sm'>
                <thead>
                  <tr className='border-b' style={{ borderColor: 'var(--semi-color-border)' }}>
                    <th className='text-left py-2 px-3 font-medium'>{t('ID')}</th>
                    <th className='text-left py-2 px-3 font-medium'>{t('标题')}</th>
                    <th className='text-left py-2 px-3 font-medium'>{t('分类')}</th>
                    <th className='text-left py-2 px-3 font-medium w-24'>{t('审计评分')}</th>
                    <th className='text-left py-2 px-3 font-medium'>{t('审计日期')}</th>
                    <th className='text-right py-2 px-3 font-medium'>{t('操作')}</th>
                  </tr>
                </thead>
                <tbody>
                  {lowScores.map((item) => (
                    <tr key={item.id} className='border-b hover:bg-gray-50' style={{ borderColor: 'var(--semi-color-border)' }}>
                      <td className='py-2 px-3'>{item.id}</td>
                      <td className='py-2 px-3 font-medium'>{item.title}</td>
                      <td className='py-2 px-3'>{item.category_name || '-'}</td>
                      <td className='py-2 px-3'>
                        <Tag color='red' size='small' shape='circle'>{item.audit_score}</Tag>
                      </td>
                      <td className='py-2 px-3 text-gray-500'>
                        {new Date(item.audit_date * 1000).toLocaleDateString()}
                      </td>
                      <td className='py-2 px-3 text-right'>
                        <Button type='tertiary' size='small' onClick={() => window.open(`/prompt/${item.id}`, '_blank')}>
                          {t('查看')}
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>
      </Spin>
    </div>
  );
};

export default SEOTrends;
