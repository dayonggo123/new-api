import { useState, useEffect, useCallback, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { API, showError } from '../../helpers';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Input,
  Tag,
  Empty,
  Spin,
  Pagination,
} from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';
import SEO from '../../components/seo/SEO';
import { WebPageSchema } from '../../components/seo/SchemaOrg';

const FALLBACK_IMAGE = 'data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 width=%22400%22 height=%22300%22%3E%3Crect width=%22400%22 height=%22300%22 fill=%22%23f0f0f0%22/%3E%3Ctext x=%2250%25%22 y=%2250%25%22 dominant-baseline=%22middle%22 text-anchor=%22middle%22 fill=%22%23999%22 font-size=%2214%22%3E%E6%9A%82%E6%97%A0%E5%9B%BE%E7%89%87%3C/text%3E%3C/svg%3E';

function parseTags(tagsStr) {
  if (!tagsStr || tagsStr === 'null') return [];
  try {
    const parsed = JSON.parse(tagsStr);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

export default function ArticleGallery() {
  const { t } = useTranslation();
  const navigate = useNavigate();

  const [articles, setArticles] = useState([]);
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(true);
  const [keyword, setKeyword] = useState('');
  const [activeCategory, setActiveCategory] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(12);
  const [total, setTotal] = useState(0);

  const loadCategories = useCallback(async () => {
    try {
      const res = await API.get('/api/public/article-categories');
      if (res.data.success) {
        setCategories(res.data.data || []);
      }
    } catch (error) {
      showError(error?.message || error);
    }
  }, []);

  const loadArticles = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      params.append('p', page);
      params.append('page_size', pageSize);
      if (keyword) params.append('keyword', keyword);
      if (activeCategory > 0) params.append('category_id', activeCategory);

      const res = await API.get(`/api/public/articles?${params.toString()}`);
      if (res.data.success) {
        setArticles(res.data.data.items || []);
        setTotal(res.data.data.total || 0);
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error?.message || error);
    }
    setLoading(false);
  }, [keyword, activeCategory, page, pageSize]);

  useEffect(() => {
    loadCategories();
  }, [loadCategories]);

  useEffect(() => {
    const timer = setTimeout(() => {
      setPage(1);
      loadArticles();
    }, 300);
    return () => clearTimeout(timer);
  }, [keyword, activeCategory]);

  useEffect(() => {
    loadArticles();
  }, [page, pageSize]);

  const getCategoryName = (categoryId) => {
    const cat = categories.find((c) => c.id === categoryId);
    return cat ? cat.name : t('未分类');
  };

  const featuredArticles = useMemo(() => articles.filter((a) => a.is_featured), [articles]);

  return (
    <div style={{ minHeight: '100vh', background: 'var(--semi-color-bg-0)', paddingTop: 64 }}>
      <SEO
        title={t('文章列表')}
        description={t('浏览和发现优质文章，涵盖 AI 技术、使用指南、行业洞察等多个领域。')}
        pathname='/article-gallery'
        keywords='AI文章, 技术博客, 使用指南, 行业洞察, AI教程'
      />
      <WebPageSchema
        pageTitle={t('文章列表')}
        pageDescription={t('浏览和发现优质文章，涵盖 AI 技术、使用指南、行业洞察等多个领域。')}
        pathname='/article-gallery'
      />

      <div style={{ maxWidth: 1200, margin: '0 auto', padding: '24px 16px' }}>
        {/* Header */}
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <h1 style={{ fontSize: 28, fontWeight: 'bold', marginBottom: 8 }}>{t('文章列表')}</h1>
          <p style={{ color: 'var(--semi-color-text-2)', fontSize: 16 }}>
            {t('浏览和发现优质文章，探索 AI 世界的无限可能。')}
          </p>
        </div>

        {/* Search & Filter */}
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12, justifyContent: 'center', marginBottom: 24 }}>
          <Input
            prefix={<IconSearch />}
            placeholder={t('搜索文章')}
            value={keyword}
            onChange={setKeyword}
            showClear
            style={{ width: 300 }}
          />
        </div>

        {/* Category Filters */}
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, justifyContent: 'center', marginBottom: 32 }}>
          <Button
            type={activeCategory === 0 ? 'primary' : 'tertiary'}
            size='small'
            onClick={() => setActiveCategory(0)}
          >
            {t('全部')}
          </Button>
          {categories.map((cat) => (
            <Button
              key={cat.id}
              type={activeCategory === cat.id ? 'primary' : 'tertiary'}
              size='small'
              onClick={() => setActiveCategory(cat.id)}
            >
              {cat.name}
            </Button>
          ))}
        </div>

        {/* Featured Articles */}
        {featuredArticles.length > 0 && activeCategory === 0 && !keyword && (
          <div style={{ marginBottom: 40 }}>
            <h2 style={{ fontSize: 20, fontWeight: 'bold', marginBottom: 16 }}>{t('精选文章')}</h2>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))', gap: 20 }}>
              {featuredArticles.map((article) => (
                <ArticleCard key={article.id} article={article} getCategoryName={getCategoryName} onClick={() => navigate(`/article/${article.id}`)} t={t} />
              ))}
            </div>
          </div>
        )}

        {/* Article Grid */}
        <Spin spinning={loading}>
          {articles.length === 0 ? (
            <Empty description={t('暂无文章')} />
          ) : (
            <>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 20 }}>
                {articles.map((article) => (
                  <ArticleCard key={article.id} article={article} getCategoryName={getCategoryName} onClick={() => navigate(`/article/${article.id}`)} t={t} />
                ))}
              </div>
              <div style={{ display: 'flex', justifyContent: 'center', marginTop: 32 }}>
                <Pagination
                  total={total}
                  pageSize={pageSize}
                  currentPage={page}
                  onPageChange={setPage}
                  showSizeChanger
                  pageSizeOpts={[12, 24, 48]}
                  onShowSizeChange={(current, size) => {
                    setPageSize(size);
                    setPage(1);
                  }}
                />
              </div>
            </>
          )}
        </Spin>
      </div>
    </div>
  );
}

function ArticleCard({ article, getCategoryName, onClick, t }) {
  const tags = parseTags(article.tags);
  const summary = article.summary || (article.content ? article.content.slice(0, 120) + '...' : '');

  return (
    <div
      onClick={onClick}
      style={{
        cursor: 'pointer',
        borderRadius: 12,
        overflow: 'hidden',
        background: 'var(--semi-color-bg-2)',
        border: '1px solid var(--semi-color-border)',
        transition: 'box-shadow 0.2s ease',
      }}
      onMouseEnter={(e) => { e.currentTarget.style.boxShadow = '0 4px 12px rgba(0,0,0,0.1)'; }}
      onMouseLeave={(e) => { e.currentTarget.style.boxShadow = 'none'; }}
    >
      <div style={{ height: 160, overflow: 'hidden', background: '#f5f5f5' }}>
        {article.media_type === 'video' && article.video_url ? (
          <video
            src={article.video_url}
            poster={article.cover_image_url || FALLBACK_IMAGE}
            style={{ width: '100%', height: '100%', objectFit: 'cover' }}
            muted
            playsInline
            preload="metadata"
          />
        ) : (
          <img
            src={article.cover_image_url || FALLBACK_IMAGE}
            alt={article.title}
            style={{ width: '100%', height: '100%', objectFit: 'cover' }}
            onError={(e) => { e.target.src = FALLBACK_IMAGE; }}
          />
        )}
      </div>
      <div style={{ padding: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
          <Tag size='small' color='light-blue'>{getCategoryName(article.category_id)}</Tag>
          {article.is_featured && <Tag size='small' color='orange'>{t('精选')}</Tag>}
        </div>
        <h3 style={{ fontSize: 16, fontWeight: 'bold', marginBottom: 8, lineHeight: 1.4 }}>
          {article.title}
        </h3>
        <p style={{ fontSize: 13, color: 'var(--semi-color-text-2)', lineHeight: 1.5, marginBottom: 12, display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', overflow: 'hidden' }}>
          {summary}
        </p>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: 12, color: 'var(--semi-color-text-3)' }}>
          <span>{article.author || '-'}</span>
          <span>{article.created_time ? new Date(article.created_time * 1000).toLocaleDateString() : ''}</span>
        </div>
        {tags.length > 0 && (
          <div style={{ marginTop: 8, display: 'flex', flexWrap: 'wrap', gap: 4 }}>
            {tags.slice(0, 3).map((tag, idx) => (
              <Tag key={idx} size='small' color='grey'>{tag}</Tag>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
