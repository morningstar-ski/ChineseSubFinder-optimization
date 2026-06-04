<template>
  <div>
    <q-list style="max-width: 600px" dense>
      <q-item tag="label">
        <q-item-section>
          <q-item-label>Assrt（https://assrt.net/api/doc）</q-item-label>
          <q-item-label caption>
            <div>注册：https://assrt.net/user/register.xml，用户面板：https://assrt.net/usercp.php</div>
            <ul class="q-pl-md">
              <li>一般用户是 5c次/min 的 API 请求限制</li>
              <li>保存后会立即生效，无需重启程序或者容器。</li>
              <li>搜索字幕效果未知，如果不用就关闭即可</li>
              <li>建议配合“保存多字幕”的选项服用（如果你使用 Emby 的话）</li>
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
            placeholder="填写你的API Token"
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
                <li>当前接入是默认关闭的薄 PoC，优先走 IMDB/TMDB 和季集参数搜索。</li>
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
              placeholder="填写你的ApiKey"
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
              placeholder="API key"
              label="OpenSubtitles API key"
              standout
              dense
              :rules="[(val) => !!val || 'Required']"
            />
          </q-item-section>
        </q-item>

        <q-item class="q-mt-sm">
          <q-item-section>
            <q-input
              :disable="!form.opensubtitles_settings.enabled"
              v-model="form.opensubtitles_settings.username"
              placeholder="Username"
              label="OpenSubtitles username"
              standout
              dense
              :rules="[(val) => !!val || 'Required']"
            />
          </q-item-section>
        </q-item>

        <q-item class="q-mt-sm">
          <q-item-section>
            <q-input
              :disable="!form.opensubtitles_settings.enabled"
              v-model="form.opensubtitles_settings.password"
              placeholder="Password"
              label="OpenSubtitles password"
              standout
              dense
              type="password"
              :rules="[(val) => !!val || 'Required']"
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

      <template v-if="form.subtitle_best_settings">
        <q-item tag="label">
          <q-item-section>
            <q-item-label>SubtitleBest</q-item-label>
            <q-item-label caption>
              <div>注册：用telegramBot注册，https://t.me/SubtitleBestBot，使用 /help 指令会有提示</div>
              <ul class="q-pl-md">
                <li>
                  此接口依赖于 IMDB ID 进行搜索，会依赖于公用的信息查询接口（获取 TMDB 、IMDB
                  等信息，如果使用人数过多，请配置自己的 TMDB API 使用）。
                </li>
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
              placeholder="填写你的ApiKey"
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
import { formModel } from 'pages/settings/use-settings';
import { toRefs } from '@vueuse/core';

const { subtitle_sources: form } = toRefs(formModel);
</script>
