<template>
  <div>
    <q-list class="settings-panel-list" dense>
      <q-item tag="label">
        <q-item-section>
          <q-item-label>Assrt（https://assrt.net/api/doc）</q-item-label>
          <q-item-label caption>
            <div>注册：https://assrt.net/user/register.xml，用户面板：https://assrt.net/usercp.php</div>
            <ul class="q-pl-md">
              <li>一般用户是 5 次/min 的 API 请求限制</li>
              <li>保存后会立即生效，无需重启程序或者容器。</li>
              <li>搜索字幕效果未知，如果不用就关闭即可。</li>
              <li>建议配合“保存多字幕”的选项使用。</li>
            </ul>
          </q-item-label>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.assrt_settings.enabled" />
        </q-item-section>
      </q-item>

      <q-item class="q-mt-sm">
        <q-item-section>
          <q-input
            :disable="!form.assrt_settings.enabled"
            v-model="form.assrt_settings.token"
            placeholder="填写你的 API Token"
            label="Assrt API Token"
            standout
            dense
            :rules="[(val) => !!val || '不能为空']"
          />
        </q-item-section>
      </q-item>

      <template v-if="form.subdl_settings">
        <q-item tag="label">
          <q-item-section>
            <q-item-label>SubDL</q-item-label>
            <q-item-label caption>
              <div>文档：https://subdl.com/api-doc</div>
              <ul class="q-pl-md">
                <li>当前接入默认关闭，优先走 IMDB/TMDB 和季集参数搜索。</li>
                <li>保存后会立即生效，无需重启程序或者容器。</li>
                <li>当前实现默认只请求中文字幕结果。</li>
              </ul>
            </q-item-label>
          </q-item-section>
          <q-item-section avatar top>
            <q-toggle v-model="form.subdl_settings.enabled" />
          </q-item-section>
        </q-item>

        <q-item class="q-mt-sm">
          <q-item-section>
            <q-input
              :disable="!form.subdl_settings.enabled"
              v-model="form.subdl_settings.key"
              placeholder="填写你的 ApiKey"
              label="SubDL ApiKey"
              standout
              dense
              :rules="[(val) => !!val || '不能为空']"
            />
          </q-item-section>
        </q-item>
      </template>

      <template v-if="form.opensubtitles_settings">
        <q-item tag="label">
          <q-item-section>
            <q-item-label>OpenSubtitles</q-item-label>
          </q-item-section>
          <q-item-section avatar top>
            <q-toggle v-model="form.opensubtitles_settings.enabled" />
          </q-item-section>
        </q-item>

        <q-item class="q-mt-sm">
          <q-item-section>
            <q-input
              :disable="!form.opensubtitles_settings.enabled"
              v-model="form.opensubtitles_settings.api_key"
              placeholder="填写 API key"
              label="OpenSubtitles API key"
              standout
              dense
              :rules="[(val) => !!val || '不能为空']"
            />
          </q-item-section>
        </q-item>

        <q-item class="q-mt-sm">
          <q-item-section>
            <q-input
              :disable="!form.opensubtitles_settings.enabled"
              v-model="form.opensubtitles_settings.username"
              placeholder="用户名"
              label="OpenSubtitles 用户名"
              standout
              dense
              :rules="[(val) => !!val || '不能为空']"
            />
          </q-item-section>
        </q-item>

        <q-item class="q-mt-sm">
          <q-item-section>
            <q-input
              :disable="!form.opensubtitles_settings.enabled"
              v-model="form.opensubtitles_settings.password"
              placeholder="密码"
              label="OpenSubtitles 密码"
              standout
              dense
              type="password"
              :rules="[(val) => !!val || '不能为空']"
            />
          </q-item-section>
        </q-item>
      </template>

      <template v-if="form.tvsubtitles_settings">
        <q-item tag="label">
          <q-item-section>
            <q-item-label>TVsubtitles</q-item-label>
          </q-item-section>
          <q-item-section avatar top>
            <q-toggle v-model="form.tvsubtitles_settings.enabled" />
          </q-item-section>
        </q-item>
      </template>

      <template v-if="form.moviesubtitles_settings">
        <q-item tag="label">
          <q-item-section>
            <q-item-label>Moviesubtitles</q-item-label>
            <q-item-label caption>
              暂时默认关闭：上游站点当前未发现可用中文字幕库存，已影响真实下载验证。
            </q-item-label>
          </q-item-section>
          <q-item-section avatar top>
            <q-toggle v-model="form.moviesubtitles_settings.enabled" />
          </q-item-section>
        </q-item>
      </template>

      <template v-if="form.subhd_settings">
        <q-item tag="label">
          <q-item-section>
            <q-item-label>SubHD</q-item-label>
          </q-item-section>
          <q-item-section avatar top>
            <q-toggle v-model="form.subhd_settings.enabled" />
          </q-item-section>
        </q-item>
      </template>

      <template v-if="form.subtitlecat_settings">
        <q-item>
          <q-item-section>
            <q-item-label>SubtitleCat</q-item-label>
            <q-item-label caption>
              英文字幕回退链默认启用 SubtitleCat，不再提供单独开关。
              <br />
              只有远端翻译中文字幕回退需要用户显式确认。
            </q-item-label>
          </q-item-section>
        </q-item>

        <q-item tag="label" class="q-mt-sm">
          <q-item-section>
            <q-item-label>SubtitleCat 中文字幕远端翻译回退</q-item-label>
            <q-item-label caption>
              默认关闭，需要手动开启。
              <br />
              仅使用 SubtitleCat 上已经生成且可直接下载的中文字幕翻译结果。
            </q-item-label>
          </q-item-section>
          <q-item-section avatar top>
            <q-toggle v-model="form.subtitlecat_settings.enable_translated_chinese_fallback" />
          </q-item-section>
        </q-item>
      </template>

      <template v-if="form.subtitle_best_settings">
        <q-item tag="label">
          <q-item-section>
            <q-item-label>SubtitleBest</q-item-label>
            <q-item-label caption>
              <div>注册：用 telegram Bot 注册，https://t.me/SubtitleBestBot，使用 /help 指令会有提示</div>
              <ul class="q-pl-md">
                <li>此接口依赖 IMDB ID 搜索，会依赖公共信息查询接口。</li>
                <li>一般用户是每天 50 次下载限制。</li>
                <li>保存后会立即生效，无需重启程序或者容器。</li>
              </ul>
            </q-item-label>
          </q-item-section>
          <q-item-section avatar top>
            <q-toggle v-model="form.subtitle_best_settings.enabled" />
          </q-item-section>
        </q-item>

        <q-item class="q-mt-sm">
          <q-item-section>
            <q-input
              :disable="!form.subtitle_best_settings.enabled"
              v-model="form.subtitle_best_settings.api_key"
              placeholder="填写你的 ApiKey"
              label="SubtitleBest ApiKey"
              standout
              dense
              :rules="[(val) => !!val || '不能为空']"
            />
          </q-item-section>
        </q-item>
      </template>
    </q-list>
  </div>
</template>

<script setup>
import { toRefs } from '@vueuse/core';
import { formModel } from 'pages/settings/use-settings';

const { subtitle_sources: form } = toRefs(formModel);
</script>
