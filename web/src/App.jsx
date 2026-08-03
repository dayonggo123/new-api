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

import React, { lazy, Suspense, useContext, useMemo } from 'react';
import { Route, Routes, useLocation, useParams, Navigate } from 'react-router-dom';
import Loading from './components/common/ui/Loading';
import User from './pages/User';
import { AuthRedirect, PrivateRoute, AdminRoute } from './helpers';
import RegisterForm from './components/auth/RegisterForm';
import LoginForm from './components/auth/LoginForm';
import NotFound from './pages/NotFound';
import Forbidden from './pages/Forbidden';
import Setting from './pages/Setting';
import { StatusContext } from './context/Status';

import PasswordResetForm from './components/auth/PasswordResetForm';
import PasswordResetConfirm from './components/auth/PasswordResetConfirm';
import Channel from './pages/Channel';
import Token from './pages/Token';
import Redemption from './pages/Redemption';
import Prompt from './pages/Prompt';
import PromptGallery from './pages/PromptGallery';
import PromptDetail from './pages/PromptDetail';
import ArticleGallery from './pages/ArticleGallery';
import ArticleDetail from './pages/ArticleDetail';
import SEOManagement from './pages/SEOManagement';
import GEOManagement from './pages/GEOManagement';
import SEOTrends from './pages/SEOTrends';
import SEOCenter from './pages/SEOCenter';
import ArticleManagement from './pages/ArticleManagement';
import ArticleEditor from './pages/ArticleEditor';
import EcommerceWizardManagement from './pages/EcommerceWizardManagement';
import ImageStudio from './pages/ImageStudio';
import NotificationManagement from './pages/NotificationManagement';
import PresetPrompt from './pages/PresetPrompt';
import SkillManagement from './pages/SkillManagement';
import SharedTemplateManagement from './pages/SharedTemplateManagement';
import AppReleaseManagement from './pages/AppReleaseManagement';
import TierManagement from './pages/TierManagement';
import TagManagement from './pages/TagManagement';
import PopupManagement from './pages/PopupManagement';
import BannerManagement from './pages/BannerManagement';
import TKMaterialManagement from './pages/TKMaterialManagement';
import TikTokVideoDownload from './pages/Tools/TikTokVideoDownload';
import TopUp from './pages/TopUp';
import Log from './pages/Log';
import Chat from './pages/Chat';
import Chat2Link from './pages/Chat2Link';
import Midjourney from './pages/Midjourney';
import Pricing from './pages/Pricing';
import Task from './pages/Task';
import ModelPage from './pages/Model';
import ModelDeploymentPage from './pages/ModelDeployment';
import Playground from './pages/Playground';
import Subscription from './pages/Subscription';
import OAuth2Callback from './components/auth/OAuth2Callback';
import PersonalSetting from './components/settings/PersonalSetting';
import Setup from './pages/Setup';
import SetupCheck from './components/layout/SetupCheck';

const Home = lazy(() => import('./pages/Home'));
const Dashboard = lazy(() => import('./pages/Dashboard'));
const About = lazy(() => import('./pages/About'));
const UserAgreement = lazy(() => import('./pages/UserAgreement'));
const PrivacyPolicy = lazy(() => import('./pages/PrivacyPolicy'));

function DynamicOAuth2Callback() {
  const { provider } = useParams();
  return <OAuth2Callback type={provider} />;
}

function App() {
  const location = useLocation();
  const [statusState] = useContext(StatusContext);

  // 获取模型广场权限配置
  const pricingRequireAuth = useMemo(() => {
    const headerNavModulesConfig = statusState?.status?.HeaderNavModules;
    if (headerNavModulesConfig) {
      try {
        const modules = JSON.parse(headerNavModulesConfig);

        // 处理向后兼容性：如果pricing是boolean，默认不需要登录
        if (typeof modules.pricing === 'boolean') {
          return false; // 默认不需要登录鉴权
        }

        // 如果是对象格式，使用requireAuth配置
        return modules.pricing?.requireAuth === true;
      } catch (error) {
        console.error('解析顶栏模块配置失败:', error);
        return false; // 默认不需要登录
      }
    }
    return false; // 默认不需要登录
  }, [statusState?.status?.HeaderNavModules]);

  return (
    <SetupCheck>
      <Routes>
        <Route
          path='/'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <Home />
            </Suspense>
          }
        />
        <Route
          path='/setup'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <Setup />
            </Suspense>
          }
        />
        <Route path='/forbidden' element={<Forbidden />} />
        <Route
          path='/console/models'
          element={
            <AdminRoute>
              <ModelPage />
            </AdminRoute>
          }
        />
        <Route
          path='/console/deployment'
          element={
            <AdminRoute>
              <ModelDeploymentPage />
            </AdminRoute>
          }
        />
        <Route
          path='/console/subscription'
          element={
            <AdminRoute>
              <Subscription />
            </AdminRoute>
          }
        />
        <Route
          path='/console/channel'
          element={
            <AdminRoute>
              <Channel />
            </AdminRoute>
          }
        />
        <Route
          path='/console/token'
          element={
            <PrivateRoute>
              <Token />
            </PrivateRoute>
          }
        />
        <Route
          path='/console/playground'
          element={
            <PrivateRoute>
              <Playground />
            </PrivateRoute>
          }
        />
        <Route
          path='/console/redemption'
          element={
            <AdminRoute>
              <Redemption />
            </AdminRoute>
          }
        />
        <Route
          path='/console/prompt'
          element={
            <AdminRoute>
              <Prompt />
            </AdminRoute>
          }
        />
        <Route
          path='/console/seo-center'
          element={
            <AdminRoute>
              <SEOCenter />
            </AdminRoute>
          }
        />
        <Route
          path='/console/seo'
          element={
            <AdminRoute>
              <Navigate to='/console/prompt' replace />
            </AdminRoute>
          }
        />
        <Route
          path='/console/article'
          element={
            <AdminRoute>
              <ArticleManagement />
            </AdminRoute>
          }
        />
        <Route
          path='/console/article/editor/:id?'
          element={
            <AdminRoute>
              <ArticleEditor />
            </AdminRoute>
          }
        />
        <Route
          path='/console/ecommerce-wizard'
          element={
            <AdminRoute>
              <EcommerceWizardManagement />
            </AdminRoute>
          }
        />
        <Route
          path='/console/geo'
          element={
            <AdminRoute>
              <GEOManagement />
            </AdminRoute>
          }
        />
        <Route
          path='/console/seo-trends'
          element={
            <AdminRoute>
              <SEOTrends />
            </AdminRoute>
          }
        />
        <Route
          path='/console/notification'
          element={
            <AdminRoute>
              <NotificationManagement />
            </AdminRoute>
          }
        />
        <Route
          path='/console/app-release'
          element={
            <AdminRoute>
              <AppReleaseManagement />
            </AdminRoute>
          }
        />
        <Route
          path='/console/preset-prompt'
          element={
            <AdminRoute>
              <PresetPrompt />
            </AdminRoute>
          }
        />
        <Route
          path='/console/skills'
          element={
            <AdminRoute>
              <SkillManagement />
            </AdminRoute>
          }
        />
        <Route
          path='/console/shared-templates'
          element={
            <AdminRoute>
              <SharedTemplateManagement />
            </AdminRoute>
          }
        />
        <Route
          path='/console/tier'
          element={
            <AdminRoute>
              <TierManagement />
            </AdminRoute>
          }
        />
        <Route
          path='/console/tag'
          element={
            <AdminRoute>
              <TagManagement />
            </AdminRoute>
          }
        />
        <Route
          path='/console/popup'
          element={
            <AdminRoute>
              <PopupManagement />
            </AdminRoute>
          }
        />
        <Route
          path='/console/banner'
          element={
            <AdminRoute>
              <BannerManagement />
            </AdminRoute>
          }
        />
        <Route
          path='/console/tk-material'
          element={
            <AdminRoute>
              <TKMaterialManagement />
            </AdminRoute>
          }
        />
        <Route
          path='/console/tools/tiktok-video-download'
          element={
            <AdminRoute>
              <TikTokVideoDownload />
            </AdminRoute>
          }
        />
        <Route
          path='/prompt-gallery'
          element={
            <PromptGallery />
          }
        />
        <Route
          path='/prompt/:id'
          element={
            <PromptDetail />
          }
        />
        <Route
          path='/article-gallery'
          element={
            <ArticleGallery />
          }
        />
        <Route
          path='/article/:id'
          element={
            <ArticleDetail />
          }
        />
        <Route
          path='/image-studio'
          element={
            <PrivateRoute>
              <ImageStudio />
            </PrivateRoute>
          }
        />
        <Route
          path='/console/user'
          element={
            <AdminRoute>
              <User />
            </AdminRoute>
          }
        />
        <Route
          path='/user/reset'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <PasswordResetConfirm />
            </Suspense>
          }
        />
        <Route
          path='/login'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <AuthRedirect>
                <LoginForm />
              </AuthRedirect>
            </Suspense>
          }
        />
        <Route
          path='/register'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <AuthRedirect>
                <RegisterForm />
              </AuthRedirect>
            </Suspense>
          }
        />
        <Route
          path='/reset'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <PasswordResetForm />
            </Suspense>
          }
        />
        <Route
          path='/oauth/github'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <OAuth2Callback type='github'></OAuth2Callback>
            </Suspense>
          }
        />
        <Route
          path='/oauth/discord'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <OAuth2Callback type='discord'></OAuth2Callback>
            </Suspense>
          }
        />
        <Route
          path='/oauth/oidc'
          element={
            <Suspense fallback={<Loading></Loading>}>
              <OAuth2Callback type='oidc'></OAuth2Callback>
            </Suspense>
          }
        />
        <Route
          path='/oauth/linuxdo'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <OAuth2Callback type='linuxdo'></OAuth2Callback>
            </Suspense>
          }
        />
        <Route
          path='/oauth/:provider'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <DynamicOAuth2Callback />
            </Suspense>
          }
        />
        <Route
          path='/console/setting'
          element={
            <AdminRoute>
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Setting />
              </Suspense>
            </AdminRoute>
          }
        />
        <Route
          path='/console/personal'
          element={
            <PrivateRoute>
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <PersonalSetting />
              </Suspense>
            </PrivateRoute>
          }
        />
        <Route
          path='/console/topup'
          element={
            <PrivateRoute>
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <TopUp />
              </Suspense>
            </PrivateRoute>
          }
        />
        <Route
          path='/console/log'
          element={
            <PrivateRoute>
              <Log />
            </PrivateRoute>
          }
        />
        <Route
          path='/console'
          element={
            <PrivateRoute>
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Dashboard />
              </Suspense>
            </PrivateRoute>
          }
        />
        <Route
          path='/console/midjourney'
          element={
            <PrivateRoute>
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Midjourney />
              </Suspense>
            </PrivateRoute>
          }
        />
        <Route
          path='/console/task'
          element={
            <PrivateRoute>
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Task />
              </Suspense>
            </PrivateRoute>
          }
        />
        <Route
          path='/pricing'
          element={
            pricingRequireAuth ? (
              <PrivateRoute>
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <Pricing />
                </Suspense>
              </PrivateRoute>
            ) : (
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Pricing />
              </Suspense>
            )
          }
        />
        <Route
          path='/about'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <About />
            </Suspense>
          }
        />
        <Route
          path='/user-agreement'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <UserAgreement />
            </Suspense>
          }
        />
        <Route
          path='/privacy-policy'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <PrivacyPolicy />
            </Suspense>
          }
        />
        <Route
          path='/console/chat/:id?'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <Chat />
            </Suspense>
          }
        />
        {/* 方便使用chat2link直接跳转聊天... */}
        <Route
          path='/chat2link'
          element={
            <PrivateRoute>
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Chat2Link />
              </Suspense>
            </PrivateRoute>
          }
        />
        <Route path='*' element={<NotFound />} />
      </Routes>
    </SetupCheck>
  );
}

export default App;
