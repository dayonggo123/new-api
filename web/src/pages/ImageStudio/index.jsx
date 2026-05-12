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

import React, { useState, useCallback } from 'react';
import {
  Button,
  Input,
  TextArea,
  Select,
  Card,
  Space,
  Typography,
  Slider,
  Tag,
} from '@douyinfe/semi-ui';
import {
  IconDownload,
  IconRefresh,
  IconImage,
  IconAIWandLevel1,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const { Title, Text } = Typography;

const DEFAULT_MODELS = [
  { label: 'DALL-E 3', value: 'dall-e-3' },
  { label: 'DALL-E 2', value: 'dall-e-2' },
];

const SIZES = [
  { label: '1024×1024', value: '1024x1024' },
  { label: '1024×1792', value: '1024x1792' },
  { label: '1792×1024', value: '1792x1024' },
  { label: '512×512', value: '512x512' },
  { label: '256×256', value: '256x256' },
];

const QUALITIES = [
  { label: 'standard', value: 'standard' },
  { label: 'hd', value: 'hd' },
];

const STYLES = [
  { label: 'vivid', value: 'vivid' },
  { label: 'natural', value: 'natural' },
];

const EXAMPLE_PROMPTS = [
  'A serene Japanese garden with cherry blossoms in full bloom, soft morning light filtering through the trees, a small stone bridge over a koi pond, watercolor painting style',
  'A futuristic cyberpunk cityscape at night, neon lights reflecting on wet streets, flying vehicles, holographic advertisements, ultra detailed digital art',
  'A cute golden retriever puppy wearing a small red scarf, sitting in a field of sunflowers, golden hour lighting, photorealistic',
  'An ancient Chinese temple perched on a misty mountain peak, traditional architecture, dramatic clouds, oil painting style',
];

const ImageStudio = () => {
  const { t } = useTranslation();
  const [prompt, setPrompt] = useState('');
  const [model, setModel] = useState('dall-e-3');
  const [customModel, setCustomModel] = useState('');
  const [size, setSize] = useState('1024x1024');
  const [n, setN] = useState(1);
  const [quality, setQuality] = useState('standard');
  const [style, setStyle] = useState('vivid');
  const [generating, setGenerating] = useState(false);
  const [results, setResults] = useState([]);
  const [history, setHistory] = useState([]);

  const handleRandomPrompt = useCallback(() => {
    const random = EXAMPLE_PROMPTS[Math.floor(Math.random() * EXAMPLE_PROMPTS.length)];
    setPrompt(random);
  }, []);

  const handleGenerate = async () => {
    if (!prompt.trim()) {
      showError(t('请输入提示词'));
      return;
    }
    setGenerating(true);
    try {
      const res = await API.post('/api/image-studio/generate', {
        model: customModel || model,
        prompt: prompt.trim(),
        n: parseInt(n) || 1,
        size,
        quality,
        style,
        response_format: 'url',
      });
      if (res.data && res.data.data && Array.isArray(res.data.data)) {
        const newImages = res.data.data.map((item) => ({
          url: item.url || item.b64_json,
          revised_prompt: item.revised_prompt || prompt,
          created: Date.now(),
          model: customModel || model,
          size,
        }));
        setResults(newImages);
        setHistory((prev) => [...newImages, ...prev].slice(0, 50));
        showSuccess(t('生成成功'));
      } else if (res.data.error) {
        showError(res.data.error.message || t('生成失败'));
      } else {
        showError(t('生成失败'));
      }
    } catch (err) {
      const msg = err.response?.data?.error?.message || err.response?.data?.message || err.message;
      showError(msg || t('生成失败'));
    }
    setGenerating(false);
  };

  const handleDownload = async (url) => {
    try {
      const response = await fetch(url);
      const blob = await response.blob();
      const blobUrl = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = blobUrl;
      a.download = `image-studio-${Date.now()}.png`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(blobUrl);
      showSuccess(t('下载成功'));
    } catch (e) {
      showError(t('下载失败'));
    }
  };

  const modelOptions = [
    ...DEFAULT_MODELS,
    ...(customModel ? [{ label: customModel, value: customModel }] : []),
  ];

  return (
    <div className='mt-[60px] px-4 pb-12 max-w-7xl mx-auto'>
      {/* Header */}
      <div className='flex items-center gap-3 mb-6'>
        <IconImage size={28} className='text-blue-500' />
        <Title heading={3} className='m-0'>{t('AI 绘画工作室')}</Title>
      </div>

      <div className='grid grid-cols-1 lg:grid-cols-12 gap-6'>
        {/* Left Panel - Parameters */}
        <div className='lg:col-span-3 space-y-4'>
          <Card className='!rounded-2xl shadow-sm border-0' bodyStyle={{ padding: 16 }}>
            <Text strong className='block mb-3'>{t('模型设置')}</Text>
            <div className='space-y-3'>
              <div>
                <Text size='small' type='tertiary' className='block mb-1'>{t('模型')}</Text>
                <Select
                  value={model}
                  onChange={setModel}
                  optionList={modelOptions}
                  style={{ width: '100%' }}
                  size='small'
                />
              </div>
              <div>
                <Text size='small' type='tertiary' className='block mb-1'>{t('自定义模型')}</Text>
                <Input
                  value={customModel}
                  onChange={setCustomModel}
                  placeholder={t('输入自定义模型名称')}
                  size='small'
                  showClear
                />
              </div>
            </div>
          </Card>

          <Card className='!rounded-2xl shadow-sm border-0' bodyStyle={{ padding: 16 }}>
            <Text strong className='block mb-3'>{t('生成参数')}</Text>
            <div className='space-y-3'>
              <div>
                <Text size='small' type='tertiary' className='block mb-1'>{t('尺寸')}</Text>
                <Select
                  value={size}
                  onChange={setSize}
                  optionList={SIZES}
                  style={{ width: '100%' }}
                  size='small'
                />
              </div>
              <div>
                <Text size='small' type='tertiary' className='block mb-1'>{t('数量')}: {n}</Text>
                <Slider
                  min={1}
                  max={4}
                  step={1}
                  value={n}
                  onChange={setN}
                  showBoundary={false}
                />
              </div>
              <div>
                <Text size='small' type='tertiary' className='block mb-1'>{t('质量')}</Text>
                <Select
                  value={quality}
                  onChange={setQuality}
                  optionList={QUALITIES}
                  style={{ width: '100%' }}
                  size='small'
                />
              </div>
              <div>
                <Text size='small' type='tertiary' className='block mb-1'>{t('风格')}</Text>
                <Select
                  value={style}
                  onChange={setStyle}
                  optionList={STYLES}
                  style={{ width: '100%' }}
                  size='small'
                />
              </div>
            </div>
          </Card>
        </div>

        {/* Center - Input & Results */}
        <div className='lg:col-span-9 space-y-4'>
          {/* Prompt Input */}
          <Card className='!rounded-2xl shadow-sm border-0' bodyStyle={{ padding: 20 }}>
            <div className='space-y-3'>
              <div className='flex justify-between items-center'>
                <Text strong>{t('提示词')}</Text>
                <Button
                  type='tertiary'
                  size='small'
                  icon={<IconRefresh />}
                  onClick={handleRandomPrompt}
                >
                  {t('随机示例')}
                </Button>
              </div>
              <TextArea
                value={prompt}
                onChange={(v) => setPrompt(v)}
                placeholder={t('描述你想生成的画面，例如：一只可爱的柴犬坐在樱花树下，阳光透过花瓣洒落，吉卜力动画风格')}
                rows={4}
                style={{ fontSize: 14 }}
                showClear
              />
              <div className='flex justify-between items-center gap-3'>
                <Text type='tertiary' size='small'>
                  {prompt.length} {t('字符')}
                </Text>
                <Button
                  theme='solid'
                  type='primary'
                  icon={<IconAIWandLevel1 />}
                  loading={generating}
                  onClick={handleGenerate}
                  size='large'
                >
                  {generating ? t('生成中...') : t('开始生成')}
                </Button>
              </div>
            </div>
          </Card>

          {/* Results */}
          {results.length > 0 && (
            <Card className='!rounded-2xl shadow-sm border-0' bodyStyle={{ padding: 20 }}>
              <div className='flex justify-between items-center mb-4'>
                <Text strong>{t('生成结果')}</Text>
                <Space>
                  <Tag color='blue'>{results.length} {t('张')}</Tag>
                  <Tag color='green'>{size}</Tag>
                </Space>
              </div>
              <div className='grid grid-cols-1 sm:grid-cols-2 gap-4'>
                {results.map((img, idx) => (
                  <div key={idx} className='group relative rounded-xl overflow-hidden border border-gray-100 bg-gray-50'>
                    <img
                      src={img.url}
                      alt={img.revised_prompt}
                      className='w-full h-auto object-cover'
                      loading='lazy'
                      onError={(e) => {
                        e.target.style.display = 'none';
                      }}
                    />
                    <div className='absolute inset-0 bg-black/0 group-hover:bg-black/40 transition-all flex items-center justify-center opacity-0 group-hover:opacity-100'>
                      <Space>
                        <Button
                          theme='solid'
                          type='primary'
                          icon={<IconDownload />}
                          onClick={() => handleDownload(img.url)}
                        >
                          {t('下载')}
                        </Button>
                      </Space>
                    </div>
                    {img.revised_prompt && img.revised_prompt !== prompt && (
                      <div className='absolute bottom-0 left-0 right-0 p-2 bg-gradient-to-t from-black/70 to-transparent opacity-0 group-hover:opacity-100 transition-opacity'>
                        <Text size='small' className='text-white line-clamp-2'>
                          {img.revised_prompt}
                        </Text>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </Card>
          )}

          {/* History */}
          {history.length > results.length && (
            <Card className='!rounded-2xl shadow-sm border-0' bodyStyle={{ padding: 20 }}>
              <Text strong className='block mb-4'>{t('历史记录')}</Text>
              <div className='grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3'>
                {history.slice(results.length).map((img, idx) => (
                  <div
                    key={idx}
                    className='relative rounded-lg overflow-hidden border border-gray-100 bg-gray-50 cursor-pointer hover:opacity-80 transition-opacity'
                    onClick={() => setResults([img])}
                  >
                    <img
                      src={img.url}
                      alt='history'
                      className='w-full aspect-square object-cover'
                      loading='lazy'
                    />
                    <div className='absolute top-1 right-1'>
                      <Tag size='small' color='white'>{img.size}</Tag>
                    </div>
                  </div>
                ))}
              </div>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
};

export default ImageStudio;
