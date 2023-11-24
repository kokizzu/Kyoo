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
using System.Linq.Expressions;
using System.Reflection;
using System.Runtime.CompilerServices;
using System.Threading.Tasks;
using Kyoo.Abstractions.Controllers;
using Kyoo.Abstractions.Models;
using Kyoo.Abstractions.Models.Attributes;
using Kyoo.Abstractions.Models.Exceptions;
using Kyoo.Abstractions.Models.Utils;
using Kyoo.Postgresql;
using Kyoo.Utils;
using Microsoft.EntityFrameworkCore;

namespace Kyoo.Core.Controllers
{
	/// <summary>
	/// A base class to create repositories using Entity Framework.
	/// </summary>
	/// <typeparam name="T">The type of this repository</typeparam>
	public abstract class LocalRepository<T> : IRepository<T>
		where T : class, IResource, IQuery
	{
		/// <summary>
		/// The Entity Framework's Database handle.
		/// </summary>
		protected DbContext Database { get; }

		/// <summary>
		/// The thumbnail manager used to store images.
		/// </summary>
		private readonly IThumbnailsManager _thumbs;

		/// <summary>
		/// Create a new base <see cref="LocalRepository{T}"/> with the given database handle.
		/// </summary>
		/// <param name="database">A database connection to load resources of type <typeparamref name="T"/></param>
		/// <param name="thumbs">The thumbnail manager used to store images.</param>
		protected LocalRepository(DbContext database, IThumbnailsManager thumbs)
		{
			Database = database;
			_thumbs = thumbs;
		}

		/// <inheritdoc/>
		public Type RepositoryType => typeof(T);

		/// <summary>
		/// Sort the given query.
		/// </summary>
		/// <param name="query">The query to sort.</param>
		/// <param name="sortBy">How to sort the query.</param>
		/// <returns>The newly sorted query.</returns>
		protected IOrderedQueryable<T> Sort(IQueryable<T> query, Sort<T>? sortBy)
		{
			sortBy ??= new Sort<T>.Default();

			IOrderedQueryable<T> _SortBy(IQueryable<T> qr, Expression<Func<T, object>> sort, bool desc, bool then)
			{
				if (then && qr is IOrderedQueryable<T> qro)
				{
					return desc
						? qro.ThenByDescending(sort)
						: qro.ThenBy(sort);
				}
				return desc
					? qr.OrderByDescending(sort)
					: qr.OrderBy(sort);
			}

			IOrderedQueryable<T> _Sort(IQueryable<T> query, Sort<T> sortBy, bool then)
			{
				switch (sortBy)
				{
					case Sort<T>.Default(var value):
						return _Sort(query, value, then);
					case Sort<T>.By(var key, var desc):
						return _SortBy(query, x => EF.Property<T>(x, key), desc, then);
					case Sort<T>.Random(var seed):
						// NOTE: To edit this, don't forget to edit the random handiling inside the KeysetPaginate function
						return _SortBy(query, x => DatabaseContext.MD5(seed + x.Id.ToString()), false, then);
					case Sort<T>.Conglomerate(var sorts):
						IOrderedQueryable<T> nQuery = _Sort(query, sorts.First(), false);
						foreach (Sort<T> sort in sorts.Skip(1))
							nQuery = _Sort(nQuery, sort, true);
						return nQuery;
					default:
						// The language should not require me to do this...
						throw new SwitchExpressionException();
				}
			}
			return _Sort(query, sortBy, false).ThenBy(x => x.Id);
		}

		protected Expression<Func<T, bool>> ParseFilter(Filter<T>? filter)
		{
			if (filter == null)
				return x => true;

			ParameterExpression x = Expression.Parameter(typeof(T), "x");

			Expression EqRandomHandler(string seed, int refId)
			{
				MethodInfo concat = typeof(string).GetMethod(nameof(string.Concat), new[] { typeof(string), typeof(string) })!;
				Expression id = Expression.Call(Expression.Property(x, "ID"), nameof(int.ToString), null);
				Expression xrng = Expression.Call(concat, Expression.Constant(seed), id);
				Expression left = Expression.Call(typeof(DatabaseContext), nameof(DatabaseContext.MD5), null, xrng);
				Expression right = Expression.Call(typeof(DatabaseContext), nameof(DatabaseContext.MD5), null, Expression.Constant($"{seed}{refId}"));
				return Expression.Equal(left, right);
			}

			BinaryExpression StringCompatibleExpression(
				Func<Expression, Expression, BinaryExpression> operand,
				Expression left,
				Expression right)
			{
				if (left.Type != typeof(string))
					return operand(left, right);
				MethodCallExpression call = Expression.Call(typeof(string), "Compare", null, left, right);
				return operand(call, Expression.Constant(0));
			}

			Expression Parse(Filter<T> f)
			{
				return f switch
				{
					Filter<T>.And(var first, var second) => Expression.AndAlso(Parse(first), Parse(second)),
					Filter<T>.Or(var first, var second) => Expression.OrElse(Parse(first), Parse(second)),
					Filter<T>.Not(var inner) => Expression.Not(Parse(inner)),
					Filter<T>.Eq(var property, var value) => Expression.Equal(Expression.Property(x, property), Expression.Constant(value)),
					Filter<T>.Ne(var property, var value) => Expression.NotEqual(Expression.Property(x, property), Expression.Constant(value)),
					Filter<T>.Gt(var property, var value) => StringCompatibleExpression(Expression.GreaterThan, Expression.Property(x, property), Expression.Constant(value)),
					Filter<T>.Ge(var property, var value) => StringCompatibleExpression(Expression.GreaterThanOrEqual, Expression.Property(x, property), Expression.Constant(value)),
					Filter<T>.Lt(var property, var value) => StringCompatibleExpression(Expression.LessThan, Expression.Property(x, property), Expression.Constant(value)),
					Filter<T>.Le(var property, var value) => StringCompatibleExpression(Expression.LessThanOrEqual, Expression.Property(x, property), Expression.Constant(value)),
					Filter<T>.Has(var property, var value) => Expression.Call(typeof(Enumerable), "Contains", new[] { value.GetType() }, Expression.Property(x, property), Expression.Constant(value)),
					Filter<T>.EqRandom(var seed, var refId) => EqRandomHandler(seed, refId),
					Filter<T>.Lambda(var lambda) => ExpressionArgumentReplacer.ReplaceParams(lambda.Body, lambda.Parameters, x),
					_ => throw new NotImplementedException(),
				};
			}

			Expression body = Parse(filter);
			return Expression.Lambda<Func<T, bool>>(body, x);
		}

		protected IQueryable<T> AddIncludes(IQueryable<T> query, Include<T>? include)
		{
			if (include == null)
				return query;
			foreach (string field in include.Fields)
				query = query.Include(field);
			return query;
		}

		/// <summary>
		/// Get a resource from it's ID and make the <see cref="Database"/> instance track it.
		/// </summary>
		/// <param name="id">The ID of the resource</param>
		/// <exception cref="ItemNotFoundException">If the item is not found</exception>
		/// <returns>The tracked resource with the given ID</returns>
		protected virtual async Task<T> GetWithTracking(int id)
		{
			T? ret = await Database.Set<T>().AsTracking().FirstOrDefaultAsync(x => x.Id == id);
			if (ret == null)
				throw new ItemNotFoundException($"No {typeof(T).Name} found with the id {id}");
			return ret;
		}

		/// <inheritdoc/>
		public virtual async Task<T> Get(int id, Include<T>? include = default)
		{
			T? ret = await GetOrDefault(id, include);
			if (ret == null)
				throw new ItemNotFoundException($"No {typeof(T).Name} found with the id {id}");
			return ret;
		}

		/// <inheritdoc/>
		public virtual async Task<T> Get(string slug, Include<T>? include = default)
		{
			T? ret = await GetOrDefault(slug, include);
			if (ret == null)
				throw new ItemNotFoundException($"No {typeof(T).Name} found with the slug {slug}");
			return ret;
		}

		/// <inheritdoc/>
		public virtual async Task<T> Get(Filter<T> filter, Include<T>? include = default)
		{
			T? ret = await GetOrDefault(filter, include: include);
			if (ret == null)
				throw new ItemNotFoundException($"No {typeof(T).Name} found with the given predicate.");
			return ret;
		}

		/// <inheritdoc />
		public virtual Task<T?> GetOrDefault(int id, Include<T>? include = default)
		{
			return AddIncludes(Database.Set<T>(), include)
				.FirstOrDefaultAsync(x => x.Id == id);
		}

		/// <inheritdoc />
		public virtual Task<T?> GetOrDefault(string slug, Include<T>? include = default)
		{
			if (slug == "random")
			{
				return AddIncludes(Database.Set<T>(), include)
					.OrderBy(x => EF.Functions.Random())
					.FirstOrDefaultAsync();
			}
			return AddIncludes(Database.Set<T>(), include)
				.FirstOrDefaultAsync(x => x.Slug == slug);
		}

		/// <inheritdoc />
		public virtual Task<T?> GetOrDefault(Filter<T>? filter,
			Include<T>? include = default,
			Sort<T>? sortBy = default)
		{
			return Sort(
					AddIncludes(Database.Set<T>(), include),
					sortBy
				)
				.FirstOrDefaultAsync(ParseFilter(filter));
		}

		/// <inheritdoc/>
		public virtual async Task<ICollection<T>> FromIds(IList<int> ids, Include<T>? include = default)
		{
			return (
				await AddIncludes(Database.Set<T>(), include)
					.Where(x => ids.Contains(x.Id))
					.ToListAsync()
				)
				.OrderBy(x => ids.IndexOf(x.Id))
				.ToList();
		}

		/// <inheritdoc/>
		public abstract Task<ICollection<T>> Search(string query, Include<T>? include = default);

		/// <inheritdoc/>
		public virtual Task<ICollection<T>> GetAll(Filter<T>? filter = null,
			Sort<T>? sort = default,
			Include<T>? include = default,
			Pagination limit = default)
		{
			return ApplyFilters(Database.Set<T>(), filter, sort, limit, include);
		}

		/// <summary>
		/// Apply filters to a query to ease sort, pagination and where queries for resources of this repository
		/// </summary>
		/// <param name="query">The base query to filter.</param>
		/// <param name="filter">An expression to filter based on arbitrary conditions</param>
		/// <param name="sort">The sort settings (sort order and sort by)</param>
		/// <param name="limit">Pagination information (where to start and how many to get)</param>
		/// <param name="include">Related fields to also load with this query.</param>
		/// <returns>The filtered query</returns>
		protected async Task<ICollection<T>> ApplyFilters(IQueryable<T> query,
			Filter<T>? filter = null,
			Sort<T>? sort = default,
			Pagination limit = default,
			Include<T>? include = default)
		{
			query = AddIncludes(query, include);
			query = Sort(query, sort);

			if (limit.AfterID != null)
			{
				T reference = await Get(limit.AfterID.Value);
				Filter<T>? keysetFilter = RepositoryHelper.KeysetPaginate(sort, reference, !limit.Reverse);
				filter = Filter.And(filter, keysetFilter);
				Console.WriteLine(filter);
			}
			if (filter != null)
				query = query.Where(ParseFilter(filter));

			if (limit.Reverse)
				query = query.Reverse();
			if (limit.Limit > 0)
				query = query.Take(limit.Limit);
			if (limit.Reverse)
				query = query.Reverse();

			return await query.ToListAsync();
		}

		/// <inheritdoc/>
		public virtual Task<int> GetCount(Filter<T>? filter = null)
		{
			IQueryable<T> query = Database.Set<T>();
			if (filter != null)
				query = query.Where(ParseFilter(filter));
			return query.CountAsync();
		}

		/// <inheritdoc/>
		public virtual async Task<T> Create(T obj)
		{
			await Validate(obj);
			if (obj is IThumbnails thumbs)
			{
				await _thumbs.DownloadImages(thumbs);
				if (thumbs.Poster != null)
					Database.Entry(thumbs).Reference(x => x.Poster).TargetEntry!.State = EntityState.Added;
				if (thumbs.Thumbnail != null)
					Database.Entry(thumbs).Reference(x => x.Thumbnail).TargetEntry!.State = EntityState.Added;
				if (thumbs.Logo != null)
					Database.Entry(thumbs).Reference(x => x.Logo).TargetEntry!.State = EntityState.Added;
			}
			return obj;
		}

		/// <inheritdoc/>
		public virtual async Task<T> CreateIfNotExists(T obj)
		{
			try
			{
				T? old = await GetOrDefault(obj.Slug);
				if (old != null)
					return old;

				return await Create(obj);
			}
			catch (DuplicatedItemException)
			{
				return await Get(obj.Slug);
			}
		}

		/// <inheritdoc/>
		public virtual async Task<T> Edit(T edited)
		{
			bool lazyLoading = Database.ChangeTracker.LazyLoadingEnabled;
			Database.ChangeTracker.LazyLoadingEnabled = false;
			try
			{
				T old = await GetWithTracking(edited.Id);

				Merger.Complete(old, edited, x => x.GetCustomAttribute<LoadableRelationAttribute>() == null);
				await EditRelations(old, edited);
				await Database.SaveChangesAsync();
				await IRepository<T>.OnResourceEdited(old);
				return old;
			}
			finally
			{
				Database.ChangeTracker.LazyLoadingEnabled = lazyLoading;
				Database.ChangeTracker.Clear();
			}
		}

		/// <inheritdoc/>
		public virtual async Task<T> Patch(int id, Func<T, Task<bool>> patch)
		{
			bool lazyLoading = Database.ChangeTracker.LazyLoadingEnabled;
			Database.ChangeTracker.LazyLoadingEnabled = false;
			try
			{
				T resource = await GetWithTracking(id);

				if (!await patch(resource))
					throw new ArgumentException("Could not patch resource");

				await Database.SaveChangesAsync();
				await IRepository<T>.OnResourceEdited(resource);
				return resource;
			}
			finally
			{
				Database.ChangeTracker.LazyLoadingEnabled = lazyLoading;
				Database.ChangeTracker.Clear();
			}
		}

		/// <summary>
		/// An overridable method to edit relation of a resource.
		/// </summary>
		/// <param name="resource">
		/// The non edited resource
		/// </param>
		/// <param name="changed">
		/// The new version of <paramref name="resource"/>.
		/// This item will be saved on the database and replace <paramref name="resource"/>
		/// </param>
		/// <returns>A <see cref="Task"/> representing the asynchronous operation.</returns>
		protected virtual Task EditRelations(T resource, T changed)
		{
			if (resource is IThumbnails thumbs && changed is IThumbnails chng)
			{
				Database.Entry(thumbs).Reference(x => x.Poster).IsModified = thumbs.Poster != chng.Poster;
				Database.Entry(thumbs).Reference(x => x.Thumbnail).IsModified = thumbs.Thumbnail != chng.Thumbnail;
				Database.Entry(thumbs).Reference(x => x.Logo).IsModified = thumbs.Logo != chng.Logo;
			}
			return Validate(resource);
		}

		/// <summary>
		/// A method called just before saving a new resource to the database.
		/// It is also called on the default implementation of <see cref="EditRelations"/>
		/// </summary>
		/// <param name="resource">The resource that will be saved</param>
		/// <exception cref="ArgumentException">
		/// You can throw this if the resource is illegal and should not be saved.
		/// </exception>
		/// <returns>A <see cref="Task"/> representing the asynchronous operation.</returns>
		protected virtual Task Validate(T resource)
		{
			if (typeof(T).GetProperty(nameof(resource.Slug))!.GetCustomAttribute<ComputedAttribute>() != null)
				return Task.CompletedTask;
			if (string.IsNullOrEmpty(resource.Slug))
				throw new ArgumentException("Resource can't have null as a slug.");
			if (int.TryParse(resource.Slug, out int _) || resource.Slug == "random")
			{
				try
				{
					MethodInfo? setter = typeof(T).GetProperty(nameof(resource.Slug))!.GetSetMethod();
					if (setter != null)
						setter.Invoke(resource, new object[] { resource.Slug + '!' });
					else
						throw new ArgumentException("Resources slug can't be number only or the literal \"random\".");
				}
				catch
				{
					throw new ArgumentException("Resources slug can't be number only or the literal \"random\".");
				}
			}
			return Task.CompletedTask;
		}

		/// <inheritdoc/>
		public virtual async Task Delete(int id)
		{
			T resource = await Get(id);
			await Delete(resource);
		}

		/// <inheritdoc/>
		public virtual async Task Delete(string slug)
		{
			T resource = await Get(slug);
			await Delete(resource);
		}

		/// <inheritdoc/>
		public virtual Task Delete(T obj)
		{
			IRepository<T>.OnResourceDeleted(obj);
			return Task.CompletedTask;
		}

		/// <inheritdoc/>
		public async Task DeleteAll(Filter<T> filter)
		{
			foreach (T resource in await GetAll(filter))
				await Delete(resource);
		}
	}
}
