import React, { useEffect, useMemo, useState } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button, Tag, Spin, Typography, Breadcrumb, Divider } from '@douyinfe/semi-ui';
import { IconArrowLeft } from '@douyinfe/semi-icons';
import { API, showError } from '../../helpers';
import SEO from '../../components/seo/SEO';
import { WebPageSchema, ArticleSchema, FAQPageSchema } from '../../components/seo/SchemaOrg';
import MarkdownRenderer from '../../components/common/markdown/MarkdownRenderer';
import HtmlRenderer from '../../components/common/HtmlRenderer';

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

/** 检测内容是否为 HTML 格式（wangEditor 输出） */
function isHtmlContent(content) {
  if (!content || typeof content !== 'string') return false;
  const trimmed = content.trim();
  // wangEditor 输出通常以 <p> 或 <div> 开头
  if (trimmed.startsWith('<p>') || trimmed.startsWith('<div>')) return true;
  // 包含大量 HTML 标签也认为是 HTML
  const htmlTagPattern = /<(p|div|span|h[1-6]|ul|ol|li|blockquote|table|tr|td|th|img|video|br)[\s>]/i;
  return htmlTagPattern.test(trimmed);
}

function applyLanguage(article, lang) {
  if (!article || lang === 'zh' || lang === 'zh-CN' || lang === 'zh-TW') return article;
  const result = { ...article };
  if (article.seo_i18n) {
    try {
      const seoMap = JSON.parse(article.seo_i18n);
      if (seoMap[lang]) {
        if (seoMap[lang].seo_title) result.seo_title = seoMap[lang].seo_title;
        if (seoMap[lang].seo_description) result.seo_description = seoMap[lang].seo_description;
        if (seoMap[lang].seo_keywords) result.seo_keywords = seoMap[lang].seo_keywords;
      }
    } catch (e) {}
  }
  if (article.i18n) {
    try {
      const contentMap = JSON.parse(article.i18n);
      if (contentMap[lang]) {
        if (contentMap[lang].title) result.title = contentMap[lang].title;
        if (contentMap[lang].summary) result.summary = contentMap[lang].summary;
        if (contentMap[lang].content) result.content = contentMap[lang].content;
      }
    } catch (e) {}
  }
  return result;
}

export default function ArticleDetail() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { t } = useTranslation();
  const [article, setArticle] = useState(null);
  const [loading, setLoading] = useState(true);
  const [activeLang, setActiveLang] = useState('zh');

  useEffect(() => {
    const urlLang = searchParams.get('lang');
    if (urlLang && SEO_LANGS.some((l) => l.code === urlLang)) {
      setActiveLang(urlLang);
    }
  }, [id, searchParams]);

  useEffect(() => {
    loadArticle();
  }, [id, activeLang]);

  const loadArticle = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (activeLang && activeLang !== 'zh') params.append('lang', activeLang);
      // 支持数字 ID 或 slug 访问
      const isNumericId = /^\d+$/.test(id);
      const apiPath = isNumericId
        ? `/api/public/articles/${id}`
        : `/api/public/articles/slug/${id}`;
      const res = await API.get(`${apiPath}?${params.toString()}`);
      const { success, data, message } = res.data;
      if (success) {
        setArticle(data);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error?.message || error);
    }
    setLoading(false);
  };

  // useMemo must be before all conditionals
  const currentArticle = useMemo(() => {
    if (!article) return null;
    return applyLanguage(article, activeLang);
  }, [article, activeLang]);

  const seoI18n = useMemo(() => {
    if (!article) return {};
    try {
      return JSON.parse(article.seo_i18n || '{}');
    } catch {
      return {};
    }
  }, [article?.seo_i18n]);

  const contentI18n = useMemo(() => {
    if (!article) return {};
    try {
      return JSON.parse(article.i18n || '{}');
    } catch {
      return {};
    }
  }, [article?.i18n]);

  const currentIntro = useMemo(() => {
    if (!currentArticle) return '';
    if (activeLang === 'zh') return currentArticle.intro || '';
    return seoI18n[activeLang]?.intro || currentArticle.intro || '';
  }, [activeLang, currentArticle?.intro, seoI18n]);

  const currentFaqList = useMemo(() => {
    if (!currentArticle) return [];
    const faqStr = activeLang === 'zh'
      ? currentArticle.faq
      : (seoI18n[activeLang]?.faq || currentArticle.faq);
    return parseFAQ(faqStr);
  }, [activeLang, currentArticle?.faq, seoI18n]);

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Spin size='large' tip={t('加载中...')} />
      </div>
    );
  }

  if (!currentArticle) {
    return (
      <div style={{ minHeight: '100vh', padding: '80px 20px', textAlign: 'center' }}>
        <Title heading={3}>{t('文章不存在')}</Title>
        <Button icon={<IconArrowLeft />} onClick={() => navigate('/article-gallery')}>
          {t('返回文章列表')}
        </Button>
      </div>
    );
  }

  const tags = parseTags(currentArticle.tags);
  const description = currentArticle.seo_description || currentArticle.summary || currentArticle.content?.slice(0, 200) || '';
  const keywords = currentArticle.seo_keywords || tags.join(', ');
  const publishedDate = currentArticle.created_time ? new Date(currentArticle.created_time * 1000).toISOString() : '';

  // 优先使用 slug 作为 URL 标识（SEO/GEO 更友好）
  const urlIdentifier = currentArticle.slug || id;
  const pagePath = `/article/${urlIdentifier}`;

  // 生成多语言 hreflang 链接
  const alternateLangs = useMemo(() => {
    return SEO_LANGS.map((lang) => ({
      lang: lang.code === 'zh' ? 'zh-CN' : lang.code,
      url: lang.code === 'zh' ? pagePath : `${pagePath}?lang=${lang.code}`,
    }));
  }, [pagePath]);

  return (
    <div style={{ minHeight: '100vh', background: 'var(--semi-color-bg-0)', paddingTop: 64 }}>
      <SEO
        title={currentArticle.seo_title || currentArticle.title}
        description={description}
        pathname={pagePath}
        keywords={keywords}
        ogImage={currentArticle.cover_image_url}
        type='article'
        alternateLangs={alternateLangs}
      />
      <WebPageSchema
        pageTitle={currentArticle.title}
        pageDescription={description}
        pathname={pagePath}
      />
      <ArticleSchema
        headline={currentArticle.title}
        description={description}
        author={currentArticle.author}
        datePublished={publishedDate}
        image={currentArticle.cover_image_url}
      />
      {currentFaqList.length > 0 && (
        <FAQPageSchema faqs={currentFaqList} />
      )}

      <div style={{ maxWidth: 800, margin: '0 auto', padding: '24px 16px' }}>
        <Breadcrumb>
          <Breadcrumb.Item onClick={() => navigate('/article-gallery')}>
            {t('文章列表')}
          </Breadcrumb.Item>
          <Breadcrumb.Item>{currentArticle.title}</Breadcrumb.Item>
        </Breadcrumb>

        <div style={{ marginTop: 24 }}>
          {/* Title */}
          <Title heading={2} style={{ marginBottom: 12 }}>
            {currentArticle.title}
          </Title>

          {/* Meta */}
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12, marginBottom: 16, color: 'var(--semi-color-text-2)', fontSize: 14 }}>
            {currentArticle.author && <span>{t('作者')}: {currentArticle.author}</span>}
            {currentArticle.created_time && (
              <span>{t('发布时间')}: {new Date(currentArticle.created_time * 1000).toLocaleDateString()}</span>
            )}
            {currentArticle.view_count > 0 && <span>{t('浏览量')}: {currentArticle.view_count}</span>}
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

          {/* Cover Image / Video */}
          {(currentArticle.cover_image_url || (currentArticle.media_type === 'video' && currentArticle.video_url)) && (
            <div style={{ marginBottom: 24 }}>
              {currentArticle.media_type === 'video' && currentArticle.video_url ? (
                <video
                  src={currentArticle.video_url}
                  poster={currentArticle.cover_image_url || FALLBACK_IMAGE}
                  controls
                  style={{ width: '100%', maxWidth: 640, borderRadius: 12, objectFit: 'cover' }}
                />
              ) : (
                <img
                  src={currentArticle.cover_image_url || FALLBACK_IMAGE}
                  alt={currentArticle.title}
                  style={{ width: '100%', maxWidth: 640, borderRadius: 12, objectFit: 'cover' }}
                  onError={(e) => { e.target.src = FALLBACK_IMAGE; }}
                />
              )}
            </div>
          )}

          {/* Intro / Summary */}
          {currentIntro && (
            <div style={{ marginBottom: 24 }}>
              <div style={{ background: '#eef2ff', padding: 16, borderRadius: 8, borderLeft: '4px solid #4f46e5' }}>
                <Text style={{ fontSize: 15, lineHeight: 1.6 }}>{currentIntro}</Text>
              </div>
            </div>
          )}
          {!currentIntro && currentArticle.summary && (
            <div style={{ marginBottom: 24, padding: 16, background: 'var(--semi-color-fill-0)', borderRadius: 8, borderLeft: '4px solid var(--semi-color-primary)' }}>
              <Text type='tertiary' size='small' style={{ fontStyle: 'italic' }}>
                {currentArticle.summary}
              </Text>
            </div>
          )}

          {/* Language Switcher */}
          <div style={{ marginBottom: 24 }}>
            <Text type='tertiary' size='small' style={{ marginBottom: 8, display: 'block' }}>
              {t('语言')}
            </Text>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
              {SEO_LANGS.map((lang) => {
                const hasTranslation = lang.code === 'zh' ||
                  (contentI18n[lang.code]?.title && contentI18n[lang.code]?.content) ||
                  !!(seoI18n[lang.code]?.intro || seoI18n[lang.code]?.faq);
                return (
                  <Button
                    key={lang.code}
                    type={activeLang === lang.code ? 'primary' : 'tertiary'}
                    size='small'
                    disabled={!hasTranslation}
                    onClick={() => setActiveLang(lang.code)}
                  >
                    {lang.label}
                  </Button>
                );
              })}
            </div>
          </div>

          {/* Content */}
          <div style={{ marginBottom: 32 }}>
            {isHtmlContent(currentArticle.content)
              ? <HtmlRenderer content={currentArticle.content} />
              : <MarkdownRenderer content={currentArticle.content || ''} />
            }
          </div>

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

          <Divider />
        </div>
      </div>
    </div>
  );
}
