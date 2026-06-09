import React, { useEffect, useMemo, useState } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button, Tag, Spin, Typography, Breadcrumb } from '@douyinfe/semi-ui';
import { IconCopy, IconArrowLeft } from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../helpers';
import SEO from '../../components/seo/SEO';
import { WebPageSchema, ArticleSchema, FAQPageSchema } from '../../components/seo/SchemaOrg';

const { Title, Text } = Typography;

const FALLBACK_IMAGE = 'data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 width=%22400%22 height=%22300%22%3E%3Crect width=%22400%22 height=%22300%22 fill=%22%23f0f0f0%22/%3E%3Ctext x=%2250%25%22 y=%2250%25%22 dominant-baseline=%22middle%22 text-anchor=%22middle%22 fill=%22%23999%22 font-size=%2214%22%3E%E6%9A%82%E6%97%A0%E5%9B%BE%E7%89%87%3C/text%3E%3C/svg%3E';

const SEO_LANGS = [
  { code: 'zh', label: '中文' },
  { code: 'en', label: 'English' },
  { code: 'fr', label: 'Français' },
  { code: 'ru', label: 'Русский' },
  { code: 'ja', label: '日本語' },
  { code: 'vi', label: 'Tiếng Việt' },
  { code: 'ko', label: '한국어' },
  { code: 'es', label: 'Español' },
  { code: 'de', label: 'Deutsch' },
  { code: 'it', label: 'Italiano' },
  { code: 'pt', label: 'Português' },
  { code: 'ar', label: 'العربية' },
];

function parseTags(tagsStr) {
  if (!tagsStr || tagsStr === 'null') return [];
  try {
    const parsed = JSON.parse(tagsStr);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function parseFAQ(faqStr) {
  if (!faqStr || faqStr === 'null') return [];
  try {
    const parsed = JSON.parse(faqStr);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

export default function PromptDetail() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { t } = useTranslation();
  const [prompt, setPrompt] = useState(null);
  const [loading, setLoading] = useState(true);
  const [activeLang, setActiveLang] = useState('zh');

  useEffect(() => {
    const urlLang = searchParams.get('lang');
    if (urlLang && SEO_LANGS.some((l) => l.code === urlLang)) {
      setActiveLang(urlLang);
    }
    loadPrompt();
  }, [id]);

  const loadPrompt = async () => {
    setLoading(true);
    try {
      // 支持数字 ID 或 slug 访问
      const isNumericId = /^\d+$/.test(id);
      const apiPath = isNumericId
        ? `/api/public/prompts/${id}`
        : `/api/public/prompts/slug/${id}`;
      const res = await API.get(apiPath);
      const { success, data, message } = res.data;
      if (success) {
        setPrompt(data);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error?.message || error);
    }
    setLoading(false);
  };

  const handleCopy = (text) => {
    navigator.clipboard.writeText(text).then(() => {
      showSuccess(t('已复制到剪贴板'));
    });
  };

  // useMemo 必须在所有条件分支之前调用
  const seoI18n = useMemo(() => {
    if (!prompt) return {};
    try {
      return JSON.parse(prompt.seo_i18n || '{}');
    } catch {
      return {};
    }
  }, [prompt?.seo_i18n]);

  const contentI18n = useMemo(() => {
    if (!prompt) return {};
    try {
      return JSON.parse(prompt.i18n || '{}');
    } catch {
      return {};
    }
  }, [prompt?.i18n]);

  const currentContent = useMemo(() => {
    if (!prompt) return '';
    if (activeLang === 'zh') return prompt.content || '';
    if (activeLang === 'en') return prompt.content_en || prompt.content || '';
    return contentI18n[activeLang]?.content || prompt.content_en || prompt.content || '';
  }, [activeLang, prompt?.content, prompt?.content_en, contentI18n]);

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Spin size='large' tip={t('加载中...')} />
      </div>
    );
  }

  if (!prompt) {
    return (
      <div style={{ minHeight: '100vh', padding: '80px 20px', textAlign: 'center' }}>
        <Title heading={3}>{t('提示词不存在')}</Title>
        <Button icon={<IconArrowLeft />} onClick={() => navigate('/prompt-gallery')}>
          {t('返回画廊')}
        </Button>
      </div>
    );
  }

  const tags = parseTags(prompt.tags);

  const currentIntro = activeLang === 'zh'
    ? prompt.intro
    : (seoI18n[activeLang]?.intro || prompt.intro);
  const currentFaqStr = activeLang === 'zh'
    ? prompt.faq
    : (seoI18n[activeLang]?.faq || prompt.faq);
  const currentFaqList = parseFAQ(currentFaqStr);
  const currentKeywords = activeLang === 'zh'
    ? (prompt.seo_keywords || tags.join(', '))
    : (seoI18n[activeLang]?.seo_keywords || prompt.seo_keywords || tags.join(', '));

  const description = currentIntro || prompt.description || prompt.content?.slice(0, 200) || '';
  const keywords = currentKeywords;

  // 优先使用 slug 作为 URL 标识（SEO/GEO 更友好）
  const urlIdentifier = prompt.slug || id;
  const pagePath = `/prompt/${urlIdentifier}`;

  // 生成多语言 hreflang 链接
  const alternateLangs = useMemo(() => {
    return SEO_LANGS.map((lang) => ({
      lang: lang.code === 'zh' ? 'zh-CN' : lang.code,
      url: lang.code === 'zh' ? pagePath : `${pagePath}?lang=${lang.code}`,
    }));
  }, [pagePath]);

  return (
    <div className='prompt-detail-page' style={{ minHeight: '100vh', background: 'var(--semi-color-bg-0)', paddingTop: 64 }}>
      <SEO
        title={prompt.title}
        description={description}
        pathname={pagePath}
        keywords={keywords}
        ogImage={prompt.cover_image_url}
        type='article'
        alternateLangs={alternateLangs}
      />
      <WebPageSchema
        pageTitle={prompt.title}
        pageDescription={description}
        pathname={pagePath}
      />
      <ArticleSchema
        headline={prompt.title}
        description={description}
        author={prompt.author || 'HarseTV'}
        datePublished={prompt.created_time ? new Date(prompt.created_time * 1000).toISOString() : ''}
        image={prompt.cover_image_url}
      />
      {currentFaqList.length > 0 && (
        <FAQPageSchema faqs={currentFaqList} />
      )}

      <div style={{ maxWidth: 800, margin: '0 auto', padding: '24px 16px' }}>
        <Breadcrumb>
          <Breadcrumb.Item onClick={() => navigate('/prompt-gallery')}>
            {t('提示词画廊')}
          </Breadcrumb.Item>
          <Breadcrumb.Item>{prompt.title}</Breadcrumb.Item>
        </Breadcrumb>

        <div style={{ marginTop: 24 }}>
          {/* Title */}
          <Title heading={2} style={{ marginBottom: 12 }}>
            {prompt.title}
          </Title>

          {/* Meta */}
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12, marginBottom: 16, color: 'var(--semi-color-text-2)', fontSize: 14 }}>
            {prompt.author && <span>{t('来源')}: {prompt.author}</span>}
            {prompt.model && <span>{t('模型')}: {prompt.model}</span>}
            <span>
              {t('类型')}: {prompt.media_type === 'video' ? t('视频') : t('图片')}
            </span>
          </div>

          {/* Tags */}
          {tags.length > 0 && (
            <div style={{ marginBottom: 20 }}>
              {tags.map((tag, idx) => (
                <Tag key={idx} color='light-blue' style={{ marginRight: 8, marginBottom: 8 }}>
                  {tag}
                </Tag>
              ))}
            </div>
          )}

          {/* 语言切换 */}
          <div style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: 2,
            borderBottom: '1px solid var(--semi-color-border)',
            marginBottom: 16,
          }}>
            {SEO_LANGS.map((lang) => {
              const hasTranslation = lang.code === 'zh'
                || !!(seoI18n[lang.code]?.intro || seoI18n[lang.code]?.faq)
                || !!(lang.code === 'en' ? prompt?.content_en : contentI18n[lang.code]?.content);
              const active = activeLang === lang.code;
              return (
                <button
                  key={lang.code}
                  type='button'
                  onClick={() => setActiveLang(lang.code)}
                  style={{
                    padding: '6px 12px',
                    border: 'none',
                    background: 'none',
                    cursor: hasTranslation ? 'pointer' : 'default',
                    borderBottom: active ? '2px solid var(--semi-color-primary)' : '2px solid transparent',
                    color: active ? 'var(--semi-color-primary)' : (hasTranslation ? 'var(--semi-color-text-2)' : '#ccc'),
                    fontWeight: active ? 600 : 400,
                    fontSize: 13,
                    transition: 'all 0.2s',
                    marginBottom: -1,
                    opacity: hasTranslation ? 1 : 0.4,
                  }}
                  disabled={!hasTranslation}
                >
                  {lang.label}
                </button>
              );
            })}
          </div>

          {/* Cover Image / Video */}
          {(prompt.cover_image_url || (prompt.media_type === 'video' && prompt.video_url)) && (
            <div style={{ marginBottom: 24 }}>
              {prompt.media_type === 'video' && prompt.video_url ? (
                <video
                  src={prompt.video_url}
                  poster={prompt.cover_image_url || FALLBACK_IMAGE}
                  controls
                  style={{ width: '100%', maxWidth: 640, borderRadius: 12 }}
                />
              ) : (
                <img
                  src={prompt.cover_image_url || FALLBACK_IMAGE}
                  alt={prompt.title}
                  style={{ width: '100%', maxWidth: 640, borderRadius: 12, objectFit: 'cover' }}
                  onError={(e) => { e.target.src = FALLBACK_IMAGE; }}
                />
              )}
            </div>
          )}

          {/* Intro */}
          {currentIntro && (
            <div style={{ marginBottom: 24 }}>
              <div style={{ background: '#eef2ff', padding: 16, borderRadius: 8, borderLeft: '4px solid #4f46e5' }}>
                <Text style={{ fontSize: 15, lineHeight: 1.6 }}>{currentIntro}</Text>
              </div>
            </div>
          )}

          {/* Content */}
          {currentContent && (
            <div style={{ marginBottom: 24 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                <Title heading={4} style={{ margin: 0 }}>
                  {activeLang === 'en' ? 'English Prompt' : activeLang === 'zh' ? t('提示词内容') : `${SEO_LANGS.find(l => l.code === activeLang)?.label || activeLang} Prompt`}
                </Title>
                <Button icon={<IconCopy />} size='small' onClick={() => handleCopy(currentContent)}>
                  {t('复制')}
                </Button>
              </div>
              <pre style={{ background: '#f8f9fa', padding: 16, borderRadius: 8, whiteSpace: 'pre-wrap', wordBreak: 'break-word', lineHeight: 1.6, fontSize: 14, border: '1px solid var(--semi-color-border)' }}>
                {currentContent}
              </pre>
            </div>
          )}

          {/* FAQ */}
          {currentFaqList.length > 0 && (
            <div style={{ marginBottom: 24 }}>
              <Title heading={4} style={{ marginBottom: 16 }}>{t('常见问题')}</Title>
              {currentFaqList.map((item, idx) => (
                <details key={idx} style={{ marginBottom: 12, padding: 12, background: '#f9fafb', borderRadius: 8, border: '1px solid var(--semi-color-border)' }}>
                  <summary style={{ fontWeight: 600, color: '#4f46e5', cursor: 'pointer', fontSize: 15 }}>
                    {item.question}
                  </summary>
                  <div style={{ marginTop: 8, color: '#555', lineHeight: 1.6 }}>
                    {item.answer}
                  </div>
                </details>
              ))}
            </div>
          )}

          {/* Back */}
          <Button icon={<IconArrowLeft />} onClick={() => navigate('/prompt-gallery')}>
            {t('返回画廊')}
          </Button>
        </div>
      </div>
    </div>
  );
}
