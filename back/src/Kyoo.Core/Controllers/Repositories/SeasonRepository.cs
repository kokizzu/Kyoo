// Kyoo - A portable and vast media library solution.
// Copyright (c) Kyoo.
//
// See AUTHORS.md and LICENSE file in the project root for full license information.
//
// Kyoo is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// any later version.
//
// Kyoo is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Kyoo. If not, see <https://www.gnu.org/licenses/>.

using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;
using Kyoo.Abstractions.Controllers;
using Kyoo.Abstractions.Models;
using Kyoo.Abstractions.Models.Utils;
using Kyoo.Postgresql;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.DependencyInjection;

namespace Kyoo.Core.Controllers
{
	/// <summary>
	/// A local repository to handle seasons.
	/// </summary>
	public class SeasonRepository : LocalRepository<Season>
	{
		/// <summary>
		/// The database handle
		/// </summary>
		private readonly DatabaseContext _database;

		/// <inheritdoc/>
		protected override Sort<Season> DefaultSort => new Sort<Season>.By(x => x.SeasonNumber);

		static SeasonRepository()
		{
			// Edit seasons slugs when the show's slug changes.
			IRepository<Show>.OnEdited += async (show) =>
			{
				await using AsyncServiceScope scope = CoreModule.Services.CreateAsyncScope();
				DatabaseContext database = scope.ServiceProvider.GetRequiredService<DatabaseContext>();
				List<Season> seasons = await database.Seasons.AsTracking()
					.Where(x => x.ShowId == show.Id)
					.ToListAsync();
				foreach (Season season in seasons)
				{
					season.ShowSlug = show.Slug;
					await database.SaveChangesAsync();
					await IRepository<Season>.OnResourceEdited(season);
				}
			};
		}

		/// <summary>
		/// Create a new <see cref="SeasonRepository"/>.
		/// </summary>
		/// <param name="database">The database handle that will be used</param>
		/// <param name="thumbs">The thumbnail manager used to store images.</param>
		public SeasonRepository(DatabaseContext database,
			IThumbnailsManager thumbs)
			: base(database, thumbs)
		{
			_database = database;
		}

		/// <inheritdoc/>
		public override async Task<ICollection<Season>> Search(string query, Include<Season>? include = default)
		{
			return await Sort(
					AddIncludes(_database.Seasons, include)
						.Where(_database.Like<Season>(x => x.Name!, $"%{query}%"))
				)
				.Take(20)
				.ToListAsync();
		}

		/// <inheritdoc/>
		public override async Task<Season> Create(Season obj)
		{
			await base.Create(obj);
			obj.ShowSlug = _database.Shows.First(x => x.Id == obj.ShowId).Slug;
			_database.Entry(obj).State = EntityState.Added;
			await _database.SaveChangesAsync(() => Get(x => x.ShowId == obj.ShowId && x.SeasonNumber == obj.SeasonNumber));
			await IRepository<Season>.OnResourceCreated(obj);
			return obj;
		}

		/// <inheritdoc/>
		protected override async Task Validate(Season resource)
		{
			await base.Validate(resource);
			if (resource.ShowId <= 0)
			{
				if (resource.Show == null)
				{
					throw new ArgumentException($"Can't store a season not related to any show " +
						$"(showID: {resource.ShowId}).");
				}
				resource.ShowId = resource.Show.Id;
			}
		}

		/// <inheritdoc/>
		public override async Task Delete(Season obj)
		{
			_database.Remove(obj);
			await _database.SaveChangesAsync();
			await base.Delete(obj);
		}
	}
}
