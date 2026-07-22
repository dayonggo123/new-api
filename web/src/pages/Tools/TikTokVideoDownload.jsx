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

import React, { useState } from 'react';
import {
  Button,
  Card,
  Col,
  Row,
  TextArea,
  Spin,
  Typography,
  Space,
  Tag,
  Table,
  Alert,
  Input,
} from '@douyinfe/semi-ui';
import { Download, Link, RefreshCw, Copy, CheckCircle } from 'lucide-react';

const { Text, Title } = Typography;
import { useTranslation } from 'react-i18next';

export default function TikTokVideoDownload() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [videoURLs, setVideoURLs] = useState('');
  const [results, setResults] = useState([]);
  const [copiedIndex, setCopiedIndex] = useState(null);

  // 解析 TikTok 分享链接，提取视频 ID 或完整 URL
  const parseTikTokURL = (url) => {
    url = url.trim();
    // 如果已经是完整的 https://www.tiktok.com/... 格式
    if (url.includes('tiktok.com')) {
      return url;
    }
    // 如果只是 ID
    return url;
  };

  // 处理下载请求
  const handleDownload = async () => {
    if (!videoURLs.trim()) {
      return;
    }

    const urls = videoURLs
      .split('\n')
      .map((url) => url.trim())
      .filter((url) => url.length > 0);

    if (urls.length === 0) {
      return;
    }

    setLoading(true);
    setResults([]);

    const newResults = [];

    for (let i = 0; i < urls.length; i++) {
      const url = urls[i];
      try {
        // 调用后端 API
        const response = await fetch(
          `/api/public/tikhub/tiktok/video-by-share-url?share_url=${encodeURIComponent(url)}`,
          {
            method: 'GET',
            headers: {
              'Content-Type': 'application/json',
            },
          }
        );

        const data = await response.json();

        if (data.success) {
          // 尝试从响应中提取视频下载链接
          let videoUrl = '';
          let coverUrl = '';
          let desc = '';
          let author = '';

          // TikHub API 返回的数据结构可能不同，尝试提取
          if (data.data) {
            const videoData = data.data;
            // 尝试多种可能的视频 URL 字段
            videoUrl =
              videoData.video?.download_addr?.url ||
              videoData.video?.play_addr?.url ||
              videoData.video_data?.play_addr?.url ||
              videoData.aweme_detail?.video?.play_addr?.url ||
              videoData.aweme_detail?.video?.download_addr?.url ||
              '';
            coverUrl =
              videoData.video?.cover?.url ||
              videoData.video_data?.cover?.url ||
              videoData.aweme_detail?.video?.cover?.url ||
              '';
            desc = videoData.desc || videoData.aweme_detail?.desc || '';
            author =
              videoData.author?.unique_id ||
              videoData.author?.nickname ||
              videoData.aweme_detail?.author?.unique_id ||
              videoData.aweme_detail?.author?.nickname ||
              '';
          }

          newResults.push({
            key: i,
            url: url,
            success: true,
            videoUrl: videoUrl,
            coverUrl: coverUrl,
            desc: desc.substring(0, 100) + (desc.length > 100 ? '...' : ''),
            author: author,
            message: videoUrl ? '获取成功' : '未找到视频地址',
          });
        } else {
          newResults.push({
            key: i,
            url: url,
            success: false,
            message: data.message || '获取失败',
          });
        }
      } catch (error) {
        newResults.push({
          key: i,
          url: url,
          success: false,
          message: error.message || '请求失败',
        });
      }
    }

    setResults(newResults);
    setLoading(false);
  };

  // 批量下载
  const handleBatchDownload = () => {
    const validResults = results.filter((r) => r.success && r.videoUrl);
    validResults.forEach((result) => {
      window.open(result.videoUrl, '_blank');
    });
  };

  // 复制下载链接
  const handleCopyLink = (videoUrl, index) => {
    navigator.clipboard.writeText(videoUrl).then(() => {
      setCopiedIndex(index);
      setTimeout(() => setCopiedIndex(null), 2000);
    });
  };

  // 清除结果
  const handleClear = () => {
    setVideoURLs('');
    setResults([]);
  };

  const columns = [
    {
      title: t('状态'),
      dataIndex: 'success',
      key: 'success',
      width: 80,
      render: (success) => (
        <Tag color={success ? 'green' : 'red'}>
          {success ? t('成功') : t('失败')}
        </Tag>
      ),
    },
    {
      title: t('输入链接'),
      dataIndex: 'url',
      key: 'url',
      ellipsis: true,
    },
    {
      title: t('视频描述'),
      dataIndex: 'desc',
      key: 'desc',
      width: 200,
      ellipsis: true,
    },
    {
      title: t('作者'),
      dataIndex: 'author',
      key: 'author',
      width: 120,
      ellipsis: true,
    },
    {
      title: t('操作'),
      key: 'action',
      width: 150,
      render: (_, record) => (
        <Space>
          {record.success && record.videoUrl && (
            <>
              <Button
                size='small'
                icon={<Copy size={14} />}
                onClick={() => handleCopyLink(record.videoUrl, record.key)}
              >
                {copiedIndex === record.key ? t('已复制') : t('复制链接')}
              </Button>
              <Button
                size='small'
                type='primary'
                icon={<Download size={14} />}
                onClick={() => window.open(record.videoUrl, '_blank')}
              >
                {t('下载')}
              </Button>
            </>
          )}
          {!record.success && (
            <Text type='danger'>{record.message}</Text>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div className='tiktok-video-download' style={{ padding: '20px' }}>
      <Row gutter={[16, 16]}>
        <Col span={24}>
          <Card>
            <Space vertical spacing={'12px'} style={{ width: '100%' }}>
              <div>
                <Title heading={5} style={{ margin: 0 }}>
                  {t('TikTok 视频无水印下载')}
                </Title>
                <Text type='secondary' style={{ fontSize: 13 }}>
                  {t('输入 TikTok 视频链接，每行一个，支持批量下载')}
                </Text>
              </div>

              <Alert
                type='info'
                style={{ maxWidth: 800 }}
                description={
                  <div>
                    <Text type='secondary'>
                      {t('支持以下格式：')}
                    </Text>
                    <ul style={{ margin: '8px 0', paddingLeft: 20 }}>
                      <li>
                        <Text type='secondary'>https://www.tiktok.com/@xxx/video/xxx</Text>
                      </li>
                      <li>
                        <Text type='secondary'>https://vm.tiktok.com/xxx</Text>
                      </li>
                      <li>
                        <Text type='secondary'>https://m.tiktok.com/v/xxx</Text>
                      </li>
                    </ul>
                  </div>
                }
              />

              <TextArea
                value={videoURLs}
                onChange={(value) => setVideoURLs(value)}
                placeholder={t('请输入 TikTok 视频链接，每行一个...')}
                rows={6}
                style={{ maxWidth: 800 }}
              />

              <Space>
                <Button
                  type='primary'
                  icon={<Download size={16} />}
                  onClick={handleDownload}
                  loading={loading}
                  disabled={!videoURLs.trim()}
                >
                  {loading ? t('处理中...') : t('获取视频')}
                </Button>
                <Button
                  icon={<RefreshCw size={16} />}
                  onClick={handleClear}
                  disabled={loading}
                >
                  {t('清除')}
                </Button>
                {results.filter((r) => r.success && r.videoUrl).length > 1 && (
                  <Button
                    type='warning'
                    icon={<Download size={16} />}
                    onClick={handleBatchDownload}
                  >
                    {t('批量下载')} (
                    {results.filter((r) => r.success && r.videoUrl).length})
                  </Button>
                )}
              </Space>
            </Space>
          </Card>
        </Col>
      </Row>

      {results.length > 0 && (
        <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
          <Col span={24}>
            <Card>
              <Title heading={5}>{t('下载结果')}</Title>
              <Table
                columns={columns}
                dataSource={results}
                pagination={false}
                style={{ marginTop: 12 }}
              />
            </Card>
          </Col>
        </Row>
      )}
    </div>
  );
}
